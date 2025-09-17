package envu

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

func IsTestBasedOnArgs() bool {
	for _, arg := range os.Args {
		if arg == "-test.v" || arg == "-test.run" || arg == "-test.timeout" || strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

func GetOrDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func MustGet[T any](key string) T {

	value, exists := os.LookupEnv(key)
	if !exists {
		panic(fmt.Sprintf("%v must be set", key))
	}
	var res T
	valueBytes := []byte(value)

	switch any(res).(type) {
	case []byte:
		return any(valueBytes).(T)
	case string:
		return any(string(valueBytes)).(T)
	default:
		// small hack to make more robust for slices
		if reflect.TypeOf(res).Kind() == reflect.Slice {
			if value[0] != '[' {
				valueBytes = append([]byte("["), valueBytes...)
			}
			if value[len(value)-1] != ']' {
				valueBytes = append(valueBytes, ']')
			}
		}
		err := json.Unmarshal(valueBytes, &res)
		if err != nil {
			panic(fmt.Errorf("envutil.MustGetOrDefault[T]: failed to unmarshal value: %v", err))
		}
		return res
	}
}

func MustGetOrDefault[T any](key string, defaultValue T) T {
	_, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return MustGet[T](key)
}

var loadDotEnvOnce = sync.Once{}

func LoadDotEnv(missingOk bool) {
	loadDotEnvOnce.Do(func() {
		f, ok := os.LookupEnv("DOT_ENV_FILE")
		// If DOT_ENV_FILE is not set, search for .env file backwards from current dir
		if !ok || f == "" {
			cwd, err := os.Getwd()
			if err != nil {
				panic(fmt.Sprintf("failed to get current directory: %v", err))
			}
			dir := cwd
			for {
				envPath := filepath.Join(dir, ".env")
				if _, err := os.Stat(envPath); err == nil {
					f = envPath
					break
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					if missingOk {
						log.Println("No .env file found")
						return
					}
					panic("no .env file found in current directory or any parent directory")
				}
				dir = parent
			}
		}
		data, err := os.ReadFile(f)
		if err != nil {
			panic(fmt.Sprintf("failed to read .env file %s: %v", f, err))
		}
		env := make(map[string]string)
		if err := parseDotEnvBytes(data, env); err != nil {
			panic(fmt.Sprintf("failed to parse .env file %s: %v", f, err))
		}
		for key, value := range env {
			// don't overload existing variables
			if _, exists := os.LookupEnv(key); !exists {
				if err := os.Setenv(key, value); err != nil {
					panic(fmt.Sprintf("failed to set environment variable %s: %v", key, err))
				}
			}
		}
	})
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
