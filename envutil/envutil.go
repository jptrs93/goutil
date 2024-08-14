package envutil

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
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
