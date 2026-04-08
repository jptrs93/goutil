package authu

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v4"
)

const csrfKey = "csrf"
const kidKey = "kid"

type activeKey struct {
	Created    time.Time
	Kid        string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type JWTAuth[T any, K any] struct {
	AutoRotateDuration time.Duration
	SigningMethod      jwt.SigningMethod
	RSAKeySize         int

	activeKey atomic.Pointer[activeKey]

	PublicKeyStorer func(kid string, key []byte) error
	PublicKeyLoader func(kid string) ([]byte, error)
	UserLoader      func(sub K) (T, error)
	mu              sync.Mutex

	CachedPublicKeys sync.Map
}

func NewJWTAuth[T any, K any](storer func(string, []byte) error, loader func(string) ([]byte, error), userLoader func(K) (T, error)) *JWTAuth[T, K] {
	return &JWTAuth[T, K]{
		AutoRotateDuration: time.Hour * 24 * 90,
		RSAKeySize:         2048,
		SigningMethod:      jwt.SigningMethodRS256,
		PublicKeyStorer:    storer,
		PublicKeyLoader:    loader,
		UserLoader:         userLoader,
		CachedPublicKeys:   sync.Map{},
	}
}

func encodeJWTSubject[K any](sub K) (string, error) {
	if s, ok := any(sub).(string); ok {
		return s, nil
	}
	b, err := json.Marshal(sub)
	if err != nil {
		return "", fmt.Errorf("encoding sub claim: %w", err)
	}
	return string(b), nil
}

func decodeJWTSubject[K any](raw string) (K, error) {
	if _, ok := any(*new(K)).(string); ok {
		return any(raw).(K), nil
	}
	var sub K
	if err := json.Unmarshal([]byte(raw), &sub); err != nil {
		return sub, fmt.Errorf("decoding sub claim: %w", err)
	}
	return sub, nil
}

func (a *JWTAuth[T, K]) GenerateTokenWith(sub K, scopes []string, ttl time.Duration) (string, error) {
	encodedSub, err := encodeJWTSubject(sub)
	if err != nil {
		return "", err
	}
	claims := jwt.MapClaims{
		"sub":    encodedSub,
		"scopes": scopes,
		"exp":    time.Now().Add(ttl).Unix(),
		"iat":    time.Now().Unix(),
	}
	return a.Sign(claims)
}

func (a *JWTAuth[T, K]) Sign(claims jwt.MapClaims) (string, error) {
	ak, err := a.ensureActiveKey()
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(a.SigningMethod, claims)
	token.Header[kidKey] = ak.Kid
	return token.SignedString(ak.PrivateKey)
}

func (a *JWTAuth[T, K]) SignDoubleSubmit(claims jwt.MapClaims) (string, string, error) {
	csrfToken, err := GenerateRandomToken(24)
	if err != nil {
		return "", "", fmt.Errorf("generating random token: %w", err)
	}
	claims[csrfKey] = csrfToken
	jwtToken, err := a.Sign(claims)
	return jwtToken, csrfToken, err
}

func (a *JWTAuth[T, K]) Verify(jwtToken string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(jwtToken, a.loadKey)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (a *JWTAuth[T, K]) VerifyAndResolveUser(jwtToken string) (jwt.MapClaims, T, error) {
	claims, err := a.Verify(jwtToken)
	if err != nil {
		var zero T
		return nil, zero, err
	}
	rawSub, ok := claims["sub"].(string)
	if !ok {
		var zero T
		return nil, zero, fmt.Errorf("missing sub claim")
	}
	sub, err := decodeJWTSubject[K](rawSub)
	if err != nil {
		var zero T
		return nil, zero, err
	}
	user, err := a.UserLoader(sub)
	if err != nil {
		var zero T
		return nil, zero, fmt.Errorf("resolving user: %w", err)
	}
	return claims, user, nil
}

func (a *JWTAuth[T, K]) VerifyDoubleSubmit(jwtToken string, headerCsrf string) (jwt.MapClaims, error) {
	claims, err := a.Verify(jwtToken)
	if err != nil {
		return nil, err
	}
	tokenCsrf, _ := claims[csrfKey].(string)
	if strings.TrimSpace(tokenCsrf) == "" || tokenCsrf != headerCsrf {
		return nil, fmt.Errorf("token csrf '%v' != header csrf '%v'", tokenCsrf, headerCsrf)
	}
	return claims, nil
}

func (a *JWTAuth[T, K]) VerifyDoubleSubmitAndResolveUser(jwtToken string, headerCsrf string) (jwt.MapClaims, T, error) {
	claims, err := a.VerifyDoubleSubmit(jwtToken, headerCsrf)
	if err != nil {
		var zero T
		return nil, zero, err
	}
	rawSub, ok := claims["sub"].(string)
	if !ok {
		var zero T
		return nil, zero, fmt.Errorf("missing sub claim")
	}
	sub, err := decodeJWTSubject[K](rawSub)
	if err != nil {
		var zero T
		return nil, zero, err
	}
	user, err := a.UserLoader(sub)
	if err != nil {
		var zero T
		return nil, zero, fmt.Errorf("resolving user: %w", err)
	}
	return claims, user, nil
}

func (a *JWTAuth[T, K]) loadKey(t *jwt.Token) (any, error) {
	kidAny, ok := t.Header[kidKey]
	if !ok {
		return nil, fmt.Errorf("missing kid header")
	}
	kid, ok := kidAny.(string)
	if !ok {
		return nil, fmt.Errorf("bad kid header value")
	}
	if ak := a.activeKey.Load(); ak != nil && kid == ak.Kid {
		return ak.PublicKey, nil
	}
	if kAny, ok := a.CachedPublicKeys.Load(kid); ok && kAny != nil {
		if verifyKey, ok := kAny.(*rsa.PublicKey); ok {
			return verifyKey, nil
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// repeat cache check incase another caller raced us
	if kAny, ok := a.CachedPublicKeys.Load(kid); ok && kAny != nil {
		if verifyKey, ok := kAny.(*rsa.PublicKey); ok {
			return verifyKey, nil
		}
	}
	b, err := a.PublicKeyLoader(kid)
	if err != nil {
		return nil, fmt.Errorf("loading verify key: %w", err)
	}
	verifyKey, err := x509.ParsePKCS1PublicKey(b)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}
	a.CachedPublicKeys.Store(kid, verifyKey)
	return verifyKey, nil
}

func (a *JWTAuth[T, K]) isActiveKeyValid(ak *activeKey) bool {
	return ak != nil && ak.PrivateKey != nil && ak.PublicKey != nil && strings.TrimSpace(ak.Kid) != "" && (a.AutoRotateDuration <= 0 || time.Since(ak.Created) < a.AutoRotateDuration)
}

func (a *JWTAuth[T, K]) ensureActiveKey() (*activeKey, error) {
	if ak := a.activeKey.Load(); a.isActiveKeyValid(ak) {
		return ak, nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Re-check after locking in case another caller already rotated the key.
	if ak := a.activeKey.Load(); a.isActiveKeyValid(ak) {
		return ak, nil
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, a.RSAKeySize)
	if err != nil {
		return nil, fmt.Errorf("generating rsa key: %w", err)
	}

	kid, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generating kid: %w", err)
	}

	publicKey := &privateKey.PublicKey
	kidStr := kid.String()
	if err := a.PublicKeyStorer(kidStr, x509.MarshalPKCS1PublicKey(publicKey)); err != nil {
		return nil, fmt.Errorf("storing public key: %w", err)
	}

	ak := &activeKey{
		Created:    time.Now(),
		Kid:        kidStr,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	a.activeKey.Store(ak)
	a.CachedPublicKeys.Store(kidStr, publicKey)

	return ak, nil
}

func GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
