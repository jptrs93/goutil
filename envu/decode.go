package envu

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
)

func Decode[T any](value string) (T, error) {
	var res T
	valueBytes := []byte(value)

	switch any(res).(type) {
	case []byte:
		return any(valueBytes).(T), nil
	case slog.Level:
		var level slog.Level
		if err := level.UnmarshalText(valueBytes); err != nil {
			return res, fmt.Errorf("envutil.MustGetOrDefault[T]: failed to unmarshal slog.Level: %v", err)
		}
		return any(level).(T), nil
	case string:
		return any(string(valueBytes)).(T), nil
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
			return res, fmt.Errorf("envutil.MustGetOrDefault[T]: failed to unmarshal value: %v", err)
		}
		return res, nil
	}
}
