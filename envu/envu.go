package envu

import (
	"fmt"
	"os"
	"strings"
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
	res, err := Decode[T](value)
	if err != nil {
		panic(err)
	}
	return res
}

func MustGetOrDefault[T any](key string, defaultValue T) T {
	_, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return MustGet[T](key)
}
