package authu

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
)

var ErrSessionInvalid = errors.New("passkey session invalid")
var ErrCredentialUnavailable = errors.New("passkey credential unavailable")
var ErrUserMismatch = errors.New("passkey user mismatch")

// WebAuthnID Users must have an opaque associated []byte ID
type WebAuthnID = []byte

type PasskeyService[U gowebauthn.User] struct {
	inner                *gowebauthn.WebAuthn
	registrationSessions inMemorySessions
	loginSessions        inMemorySessions
	SaveCredential       func(userID WebAuthnID, credential *gowebauthn.Credential) error
	LoadUser             func(userID WebAuthnID) (U, error)
	LoadCredentialOwner  func(credentialID WebAuthnID) (user U, err error)
	SessionTTL           time.Duration
}

type passkeySession struct {
	SessionID string
	UserID    []byte
	HasUserID bool
	ExpiresAt time.Time
	Payload   []byte
}

type inMemorySessions struct {
	mu      sync.Mutex
	records map[string]passkeySession
}

func NewPasskeyService[U gowebauthn.User](config *gowebauthn.Config, saveCredential func(userID []byte, credential *gowebauthn.Credential) error, loadUser func(userID []byte) (U, error), loadCredentialOwner func(credentialID []byte) (user U, err error)) (*PasskeyService[U], error) {
	if config == nil {
		return nil, fmt.Errorf("missing webauthn config")
	}
	if saveCredential == nil || loadUser == nil || loadCredentialOwner == nil {
		return nil, fmt.Errorf("missing passkey dependencies")
	}
	inner, err := gowebauthn.New(config)
	if err != nil {
		return nil, fmt.Errorf("creating webauthn config: %w", err)
	}
	return &PasskeyService[U]{
		inner:                inner,
		registrationSessions: inMemorySessions{records: map[string]passkeySession{}},
		loginSessions:        inMemorySessions{records: map[string]passkeySession{}},
		SaveCredential:       saveCredential,
		LoadUser:             loadUser,
		LoadCredentialOwner:  loadCredentialOwner,
		SessionTTL:           5 * time.Minute,
	}, nil
}

func (s *PasskeyService[U]) BeginRegistration(userID []byte) (string, []byte, error) {
	user, err := s.LoadUser(userID)
	if err != nil {
		return "", nil, err
	}
	creation, session, err := s.inner.BeginRegistration(
		user,
		gowebauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		gowebauthn.WithExclusions(gowebauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		gowebauthn.WithExtensions(map[string]any{"credProps": true}),
	)
	if err != nil {
		return "", nil, err
	}
	optionsJSON, err := json.Marshal(creation)
	if err != nil {
		return "", nil, fmt.Errorf("marshal registration options: %w", err)
	}
	record, err := s.newSession(userID, true, session)
	if err != nil {
		return "", nil, err
	}
	s.registrationSessions.save(record)
	return record.SessionID, optionsJSON, nil
}

func (s *PasskeyService[U]) FinishRegistration(userID []byte, sessionID string, credentialJSON []byte) ([]byte, error) {
	record, err := s.consumeSession(&s.registrationSessions, sessionID)
	if err != nil {
		return nil, err
	}
	if !record.HasUserID || !bytes.Equal(record.UserID, userID) {
		return nil, ErrUserMismatch
	}
	user, err := s.LoadUser(userID)
	if err != nil {
		return nil, err
	}
	session, err := decodeSession(record.Payload)
	if err != nil {
		return nil, err
	}
	credential, err := s.inner.FinishRegistration(user, *session, httpRequestWithBody(credentialJSON))
	if err != nil {
		return nil, err
	}
	if err := s.saveCredential(userID, credential); err != nil {
		return nil, err
	}
	return credential.ID, nil
}

func (s *PasskeyService[U]) BeginLogin() (string, []byte, error) {
	assertion, session, err := s.inner.BeginDiscoverableLogin(gowebauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return "", nil, err
	}
	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return "", nil, fmt.Errorf("marshal login options: %w", err)
	}
	record, err := s.newSession(nil, false, session)
	if err != nil {
		return "", nil, err
	}
	s.loginSessions.save(record)
	return record.SessionID, optionsJSON, nil
}

func (s *PasskeyService[U]) FinishLogin(sessionID string, credentialJSON []byte) (U, error) {
	var zero U
	record, err := s.consumeSession(&s.loginSessions, sessionID)
	if err != nil {
		return zero, err
	}
	session, err := decodeSession(record.Payload)
	if err != nil {
		return zero, err
	}
	var resolvedUserID []byte
	validatedUser, credential, err := s.inner.FinishPasskeyLogin(func(rawID []byte, userHandle []byte) (gowebauthn.User, error) {
		user, err := s.LoadCredentialOwner(rawID)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(userHandle, user.WebAuthnID()) {
			return nil, ErrCredentialUnavailable
		}
		resolvedUserID = bytes.Clone(user.WebAuthnID())
		return user, nil
	}, *session, httpRequestWithBody(credentialJSON))
	if err != nil {
		return zero, err
	}
	user, ok := validatedUser.(U)
	if !ok {
		return zero, fmt.Errorf("unexpected passkey user type %T", validatedUser)
	}
	if err := s.saveCredential(resolvedUserID, credential); err != nil {
		return zero, err
	}
	return user, nil
}

func (s *PasskeyService[U]) newSession(userID []byte, hasUserID bool, session *gowebauthn.SessionData) (passkeySession, error) {
	sessionID, err := GenerateRandomToken(24)
	if err != nil {
		return passkeySession{}, fmt.Errorf("generate passkey session id: %w", err)
	}
	payload, err := encodeSession(session)
	if err != nil {
		return passkeySession{}, err
	}
	expiresAt := session.Expires
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(s.sessionTTL())
	}
	return passkeySession{
		SessionID: sessionID,
		UserID:    bytes.Clone(userID),
		HasUserID: hasUserID,
		ExpiresAt: expiresAt,
		Payload:   payload,
	}, nil
}

func (s *PasskeyService[U]) consumeSession(store *inMemorySessions, sessionID string) (*passkeySession, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrSessionInvalid
	}
	return store.consume(sessionID)
}

func (s *PasskeyService[U]) saveCredential(userID []byte, credential *gowebauthn.Credential) error {
	return s.SaveCredential(userID, credential)
}

func (s *PasskeyService[U]) sessionTTL() time.Duration {
	if s.SessionTTL <= 0 {
		return 5 * time.Minute
	}
	return s.SessionTTL
}

func (s *inMemorySessions) save(record passkeySession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteExpiredLocked(time.Now())
	s.records[record.SessionID] = record
}

func (s *inMemorySessions) consume(sessionID string) (*passkeySession, error) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteExpiredLocked(now)
	record, ok := s.records[sessionID]
	if !ok {
		return nil, ErrSessionInvalid
	}
	delete(s.records, sessionID)
	if now.After(record.ExpiresAt) {
		return nil, ErrSessionInvalid
	}
	return &record, nil
}

func (s *inMemorySessions) deleteExpiredLocked(now time.Time) {
	for sessionID, record := range s.records {
		if now.After(record.ExpiresAt) {
			delete(s.records, sessionID)
		}
	}
}

func encodeSession(session *gowebauthn.SessionData) ([]byte, error) {
	if session == nil {
		return nil, fmt.Errorf("nil passkey session")
	}
	return session.MarshalMsg(nil)
}

func decodeSession(b []byte) (*gowebauthn.SessionData, error) {
	var session gowebauthn.SessionData
	if _, err := session.UnmarshalMsg(b); err != nil {
		return nil, fmt.Errorf("decode passkey session: %w", err)
	}
	return &session, nil
}

func httpRequestWithBody(body []byte) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GenerateWebAuthnID(n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("webauthn id length must be positive")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate webauthn id: %w", err)
	}
	return b, nil
}
