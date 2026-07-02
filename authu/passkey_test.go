package authu

import (
	"encoding/json"
	"testing"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type passkeyTestUser struct{}

func (passkeyTestUser) WebAuthnID() []byte                         { return []byte("test-user") }
func (passkeyTestUser) WebAuthnName() string                       { return "test-user" }
func (passkeyTestUser) WebAuthnDisplayName() string                { return "Test User" }
func (passkeyTestUser) WebAuthnCredentials() []webauthn.Credential { return nil }

func TestPasskeyServiceBeginLoginUsesConfiguredUserVerification(t *testing.T) {
	service := newPasskeyTestService(t, protocol.VerificationRequired)

	_, optionsJSON, err := service.BeginLogin()
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	if got := loginOptionsUserVerification(t, optionsJSON); got != protocol.VerificationRequired {
		t.Fatalf("login userVerification = %q, want %q", got, protocol.VerificationRequired)
	}
}

func TestPasskeyServiceBeginLoginDefaultsToPreferredUserVerification(t *testing.T) {
	service := newPasskeyTestService(t, "")

	_, optionsJSON, err := service.BeginLogin()
	if err != nil {
		t.Fatalf("BeginLogin: %v", err)
	}

	if got := loginOptionsUserVerification(t, optionsJSON); got != protocol.VerificationPreferred {
		t.Fatalf("login userVerification = %q, want %q", got, protocol.VerificationPreferred)
	}
}

func newPasskeyTestService(t *testing.T, userVerification protocol.UserVerificationRequirement) *PasskeyService[*passkeyTestUser] {
	t.Helper()
	service, err := NewPasskeyService[*passkeyTestUser](&webauthn.Config{
		RPDisplayName: "Test RP",
		RPID:          "example.com",
		RPOrigins:     []string{"https://example.com"},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: userVerification,
		},
	}, func([]byte, *webauthn.Credential) error {
		return nil
	}, func([]byte) (*passkeyTestUser, error) {
		return &passkeyTestUser{}, nil
	}, func([]byte) (*passkeyTestUser, error) {
		return &passkeyTestUser{}, nil
	})
	if err != nil {
		t.Fatalf("NewPasskeyService: %v", err)
	}
	return service
}

func loginOptionsUserVerification(t *testing.T, optionsJSON []byte) protocol.UserVerificationRequirement {
	t.Helper()
	var options struct {
		PublicKey struct {
			UserVerification protocol.UserVerificationRequirement `json:"userVerification"`
		} `json:"publicKey"`
	}
	if err := json.Unmarshal(optionsJSON, &options); err != nil {
		t.Fatalf("unmarshal login options: %v", err)
	}
	return options.PublicKey.UserVerification
}
