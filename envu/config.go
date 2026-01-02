package envu

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
)

func MustLoadConfig[T any](overridePrefix string) T {
	c, err := LoadConfig[T](overridePrefix)
	if err != nil {
		panic(err)
	}
	return c
}

func LoadConfig[T any](overridePrefix string) (T, error) {
	var config T
	v := reflect.ValueOf(&config).Elem()
	t := v.Type()

	slogLevelType := reflect.TypeOf(slog.Level(0))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		envTag := field.Tag.Get("env")
		if envTag == "" {
			continue
		}

		parts := strings.SplitN(envTag, ",", 2)
		envVarName := parts[0]

		// Get the string value from environment or default
		var value string
		if overridePrefix != "" {
			// if there is a value for the override prefix (like 'STAGING_') then use that.
			value = os.Getenv(overridePrefix + envVarName)
		}
		if value == "" {
			value = os.Getenv(envVarName)
			if value == "" && len(parts) > 1 {
				value = parts[1]
			} else if value == "" {
				return config, fmt.Errorf("env var %v missing", envVarName)
			}
		}

		// Decode the string value to the appropriate type
		switch {
		case field.Type == slogLevelType:
			level := decodeLogLevel(value)
			fieldValue.Set(reflect.ValueOf(level))
		case field.Type.Kind() == reflect.String:
			fieldValue.SetString(value)
		case field.Type.Kind() == reflect.Int, field.Type.Kind() == reflect.Int8,
			field.Type.Kind() == reflect.Int16, field.Type.Kind() == reflect.Int32,
			field.Type.Kind() == reflect.Int64:
			decoded, err := Decode[int64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as int: %v", envVarName, err)
			}
			fieldValue.SetInt(decoded)
		case field.Type.Kind() == reflect.Uint, field.Type.Kind() == reflect.Uint8,
			field.Type.Kind() == reflect.Uint16, field.Type.Kind() == reflect.Uint32,
			field.Type.Kind() == reflect.Uint64:
			decoded, err := Decode[uint64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as uint: %v", envVarName, err)
			}
			fieldValue.SetUint(decoded)
		case field.Type.Kind() == reflect.Float32, field.Type.Kind() == reflect.Float64:
			decoded, err := Decode[float64](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as float: %v", envVarName, err)
			}
			fieldValue.SetFloat(decoded)
		case field.Type.Kind() == reflect.Bool:
			decoded, err := Decode[bool](value)
			if err != nil {
				return config, fmt.Errorf("failed to decode %v as bool: %v", envVarName, err)
			}
			fieldValue.SetBool(decoded)
		default:
			// For complex types (slices, structs, etc.), use reflection to decode
			result := reflect.ValueOf(Decode[any]).Call([]reflect.Value{reflect.ValueOf(value)})
			if !result[1].IsNil() {
				return config, fmt.Errorf("failed to decode %v: %v", envVarName, result[1].Interface().(error))
			}

			// Convert the result to the correct type and set it
			convertedValue := result[0].Convert(field.Type)
			fieldValue.Set(convertedValue)
		}
	}

	return config, nil
}

func decodeLogLevel(logLevelStr string) slog.Level {
	var level slog.Level
	err := json.Unmarshal([]byte(fmt.Sprintf("\"%s\"", logLevelStr)), &level)
	if err == nil {
		return level
	}
	slog.Warn(fmt.Sprintf("failed decoding log level str '%v' (defaulting to INFO): %v", logLevelStr, err))
	return slog.LevelInfo
}
