package envu

import "testing"

func TestIsValidPOSIXEnvKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{key: "A", want: true},
		{key: "_", want: true},
		{key: "APP_NAME", want: true},
		{key: "app_name", want: true},
		{key: "APP_NAME_1", want: true},
		{key: "", want: false},
		{key: "1_APP_NAME", want: false},
		{key: "APP-NAME", want: false},
		{key: "APP.NAME", want: false},
		{key: "APP NAME", want: false},
		{key: "APP=NAME", want: false},
		{key: "APP_NAME!", want: false},
		{key: "APP_NAMÉ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := IsValidPOSIXEnvKey(tt.key)
			if got != tt.want {
				t.Fatalf("IsValidPOSIXEnvKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}
