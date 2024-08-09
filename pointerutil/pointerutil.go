package pointerutil

import "reflect"

func SafeDref[T any](v *T) T {
	var result T
	if v == nil {
		return result
	}
	return *v
}

func NilRef[T any](v T) *T {
	if IsZeroValue(v) {
		return nil
	}
	return &v
}

func NonNil[T any](v *T, d T) T {
	if v == nil {
		return d
	}
	return *v
}

func IsTrue(b *bool) bool {
	return b != nil && *b
}

func To[T any](t T) *T {
	return &t
}

func IsZeroValue(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return x == reflect.Zero(reflect.TypeOf(x)).Interface()
	}
}
