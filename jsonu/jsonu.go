package jsonu

import (
	"encoding/json"
	"fmt"
)

func Unmarshal[T any](b []byte) (T, error) {
	var res T
	if err := json.Unmarshal(b, &res); err != nil {
		return res, fmt.Errorf("unmarshalling %T: %w", res, err)
	}
	return res, nil
}
