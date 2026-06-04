package envu

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
)

func MustGetS(key string, seed int64) string {
	v, err := GetS(key, seed)
	if err != nil {
		panic(err)
	}
	return v
}

func GetS(key string, seed int64) (string, error) {
	encryptedValue := os.Getenv(xorC(key, seed))
	if encryptedValue == "" {
		return "", fmt.Errorf("environment variable not found")
	}
	return xorD(encryptedValue, seed)
}

func EncodeS(key, value string, seed int64) (string, string, error) {
	encryptedKey := xorC(key, seed)
	encryptedValue := xorC(value, seed)
	return encryptedKey, encryptedValue, nil
}

func xorC(text string, seed int64) string {
	r := rand.New(rand.NewSource(seed))
	b := []byte(text)
	result := make([]byte, len(b))
	for i := range b {
		result[i] = b[i] ^ byte(r.Intn(256))
	}
	return base64.RawURLEncoding.EncodeToString(result)
}

func xorD(encoded string, seed int64) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	r := rand.New(rand.NewSource(seed))
	result := make([]byte, len(data))
	for i := range data {
		result[i] = data[i] ^ byte(r.Intn(256))
	}
	return string(result), nil
}
