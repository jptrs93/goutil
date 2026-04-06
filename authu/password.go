package authu

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"golang.org/x/crypto/argon2"
)

const (
	pMemory     = 64 * 1024 // 64 MB
	pTime       = 3         // iterations
	pSaltLength = 16
	pKeyLength  = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, pSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, pTime, pMemory, 1, pKeyLength)
	// Concatenate salt + hash
	combined := append(salt, hash...)
	return base64.RawStdEncoding.EncodeToString(combined), nil
}

func VerifyPassword(password, passwordHash string) (bool, error) {
	combined, err := base64.RawStdEncoding.DecodeString(passwordHash)
	if err != nil {
		return false, err
	}
	if len(combined) != pSaltLength+pKeyLength {
		return false, errors.New("invalid hash length")
	}
	salt := combined[:pSaltLength]
	hash := combined[pSaltLength:]
	computedHash := argon2.IDKey([]byte(password), salt, pTime, pMemory, 1, pKeyLength)
	return subtle.ConstantTimeCompare(computedHash, hash) == 1, nil
}
