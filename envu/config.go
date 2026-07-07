package envu

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/jptrs93/goutil/erru"
)

func MustParse[T any](loadEnvValueFunc func(k string) (string, bool)) T {
	return erru.Must(Parse[T](loadEnvValueFunc))
}

func Parse[T any](loadEnvValueFunc func(k string) (string, bool)) (T, error) {
	var config T
	v := reflect.ValueOf(&config).Elem()
	if err := parseFields(v, loadEnvValueFunc); err != nil {
		return config, err
	}
	return config, nil
}

func parseFields(v reflect.Value, loadEnvValueFunc func(k string) (string, bool)) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if field.PkgPath != "" {
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag == "" {
			if field.Anonymous && fieldValue.Kind() == reflect.Struct {
				if err := parseFields(fieldValue, loadEnvValueFunc); err != nil {
					return err
				}
			}
			continue
		}
		parts := strings.SplitN(envTag, ",", 2)
		envVarName := parts[0]
		value, ok := loadEnvValueFunc(envVarName)
		if !ok {
			if len(parts) > 1 {
				value = parts[1]
			} else if field.Type.Kind() == reflect.Pointer {
				continue
			} else {
				return fmt.Errorf("required env var %v missing", envVarName)
			}
		}
		if err := setConfigField(fieldValue, value, envVarName); err != nil {
			return err
		}
	}
	return nil
}

func setConfigField(fieldValue reflect.Value, value string, envVarName string) error {
	fieldType := fieldValue.Type()
	if fieldType.Kind() == reflect.Pointer {
		pointerValue := reflect.New(fieldType.Elem())
		if err := setConfigField(pointerValue.Elem(), value, envVarName); err != nil {
			return err
		}
		fieldValue.Set(pointerValue)
		return nil
	}

	slogLevelType := reflect.TypeOf(slog.Level(0))
	switch {
	case fieldType == slogLevelType:
		level := decodeLogLevel(value)
		fieldValue.Set(reflect.ValueOf(level))
	case fieldType.Kind() == reflect.String:
		fieldValue.SetString(value)
	case fieldType.Kind() == reflect.Int, fieldType.Kind() == reflect.Int8,
		fieldType.Kind() == reflect.Int16, fieldType.Kind() == reflect.Int32,
		fieldType.Kind() == reflect.Int64:
		decoded, err := Decode[int64](value)
		if err != nil {
			return fmt.Errorf("failed to decode %v as int: %v", envVarName, err)
		}
		fieldValue.SetInt(decoded)
	case fieldType.Kind() == reflect.Uint, fieldType.Kind() == reflect.Uint8,
		fieldType.Kind() == reflect.Uint16, fieldType.Kind() == reflect.Uint32,
		fieldType.Kind() == reflect.Uint64:
		decoded, err := Decode[uint64](value)
		if err != nil {
			return fmt.Errorf("failed to decode %v as uint: %v", envVarName, err)
		}
		fieldValue.SetUint(decoded)
	case fieldType.Kind() == reflect.Float32, fieldType.Kind() == reflect.Float64:
		decoded, err := Decode[float64](value)
		if err != nil {
			return fmt.Errorf("failed to decode %v as float: %v", envVarName, err)
		}
		fieldValue.SetFloat(decoded)
	case fieldType.Kind() == reflect.Bool:
		decoded, err := Decode[bool](value)
		if err != nil {
			return fmt.Errorf("failed to decode %v as bool: %v", envVarName, err)
		}
		fieldValue.SetBool(decoded)
	case fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.String:
		parts := strings.Split(value, ",")
		trimmed := make([]string, 0, len(parts))
		for _, part := range parts {
			item := strings.TrimSpace(part)
			if item != "" {
				trimmed = append(trimmed, item)
			}
		}
		fieldValue.Set(reflect.ValueOf(trimmed))
	default:
		result := reflect.ValueOf(Decode[any]).Call([]reflect.Value{reflect.ValueOf(value)})
		if !result[1].IsNil() {
			return fmt.Errorf("failed to decode %v: %v", envVarName, result[1].Interface().(error))
		}

		convertedValue := result[0].Convert(fieldType)
		fieldValue.Set(convertedValue)
	}

	return nil
}

func decodeLogLevel(logLevelStr string) slog.Level {
	level, err := Decode[slog.Level](logLevelStr)
	if err == nil {
		return level
	}
	slog.Warn(fmt.Sprintf("failed decoding log level str '%v' (defaulting to INFO): %v", logLevelStr, err))
	return slog.LevelInfo
}
