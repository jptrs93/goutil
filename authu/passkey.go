package authu

import (
	"bytes"
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

type User interface {
	gowebauthn.User
	ID() string
}

type UserResolver interface {
	LoadByID(id string) (User, error)
}

type CredentialStore interface {
	Save(userID string, credential *gowebauthn.Credential) error
	Fetch(credentialID []byte) (userID string, credential *gowebauthn.Credential, err error)
	ListByUserID(userID string) ([]gowebauthn.Credential, error)
}

type PasskeyService struct {
	inner                *gowebauthn.WebAuthn
	registrationSessions inMemorySessions
	loginSessions        inMemorySessions
	credentials          CredentialStore
	users                UserResolver
	SessionTTL           time.Duration
}

type passkeySession struct {
	SessionID string
	UserID    string
	ExpiresAt time.Time
	Payload   []byte
}

type inMemorySessions struct {
	mu      sync.Mutex
	records map[string]passkeySession
}

func NewPasskeyService(config *gowebauthn.Config, credentials CredentialStore, users UserResolver) (*PasskeyService, error) {
	if config == nil {
		return nil, fmt.Errorf("missing webauthn config")
	}
	if credentials == nil || users == nil {
		return nil, fmt.Errorf("missing passkey dependencies")
	}
	inner, err := gowebauthn.New(config)
	if err != nil {
		return nil, fmt.Errorf("creating webauthn config: %w", err)
	}
	return &PasskeyService{
		inner:                inner,
		registrationSessions: inMemorySessions{records: map[string]passkeySession{}},
		loginSessions:        inMemorySessions{records: map[string]passkeySession{}},
		credentials:          credentials,
		users:                users,
		SessionTTL:           5 * time.Minute,
	}, nil
}

func (s *PasskeyService) BeginRegistration(userID string) (string, []byte, error) {
	user, err := s.users.LoadByID(userID)
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
	record, err := s.newSession(userID, session)
	if err != nil {
		return "", nil, err
	}
	s.registrationSessions.save(record)
	return record.SessionID, optionsJSON, nil
}

func (s *PasskeyService) FinishRegistration(userID string, sessionID string, credentialJSON []byte) ([]byte, error) {
	record, err := s.consumeSession(&s.registrationSessions, sessionID)
	if err != nil {
		return nil, err
	}
	if record.UserID != userID {
		return nil, ErrUserMismatch
	}
	user, err := s.users.LoadByID(userID)
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

func (s *PasskeyService) BeginLogin() (string, []byte, error) {
	assertion, session, err := s.inner.BeginDiscoverableLogin(gowebauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return "", nil, err
	}
	optionsJSON, err := json.Marshal(assertion)
	if err != nil {
		return "", nil, fmt.Errorf("marshal login options: %w", err)
	}
	record, err := s.newSession("", session)
	if err != nil {
		return "", nil, err
	}
	s.loginSessions.save(record)
	return record.SessionID, optionsJSON, nil
}

func (s *PasskeyService) FinishLogin(sessionID string, credentialJSON []byte) (User, error) {
	record, err := s.consumeSession(&s.loginSessions, sessionID)
	if err != nil {
		return nil, err
	}
	session, err := decodeSession(record.Payload)
	if err != nil {
		return nil, err
	}
	validatedUser, credential, err := s.inner.FinishPasskeyLogin(s.loadLoginUser, *session, httpRequestWithBody(credentialJSON))
	if err != nil {
		return nil, err
	}
	user, ok := validatedUser.(User)
	if !ok {
		return nil, fmt.Errorf("unexpected passkey user type %T", validatedUser)
	}
	if err := s.saveCredential(user.ID(), credential); err != nil {
		return nil, err
	}
	return user, nil
}

func EncodeCredential(credential *gowebauthn.Credential) ([]byte, error) {
	if credential == nil {
		return nil, fmt.Errorf("nil passkey credential")
	}
	return json.Marshal(credential)
}

func DecodeCredential(b []byte) (*gowebauthn.Credential, error) {
	var credential gowebauthn.Credential
	if err := json.Unmarshal(b, &credential); err != nil {
		return nil, fmt.Errorf("decode passkey credential: %w", err)
	}
	return &credential, nil
}

func (s *PasskeyService) newSession(userID string, session *gowebauthn.SessionData) (passkeySession, error) {
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
		UserID:    userID,
		ExpiresAt: expiresAt,
		Payload:   payload,
	}, nil
}

func (s *PasskeyService) consumeSession(store *inMemorySessions, sessionID string) (*passkeySession, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrSessionInvalid
	}
	return store.consume(sessionID)
}

func (s *PasskeyService) loadLoginUser(rawID []byte, userHandle []byte) (gowebauthn.User, error) {
	userID, _, err := s.credentials.Fetch(rawID)
	if err != nil {
		return nil, err
	}
	user, err := s.users.LoadByID(userID)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(userHandle, user.WebAuthnID()) {
		return nil, ErrCredentialUnavailable
	}
	return user, nil
}

func (s *PasskeyService) saveCredential(userID string, credential *gowebauthn.Credential) error {
	return s.credentials.Save(userID, credential)
}

func (s *PasskeyService) sessionTTL() time.Duration {
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
