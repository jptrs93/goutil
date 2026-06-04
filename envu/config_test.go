package envu

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	type config struct {
		AppName      string   `env:"APP_NAME"`
		AppEnv       string   `env:"APP_ENV,production"`
		Port         int      `env:"PORT"`
		Debug        bool     `env:"DEBUG"`
		RateLimit    float64  `env:"RATE_LIMIT"`
		AllowedHosts []string `env:"ALLOWED_HOSTS"`
		Ignored      string
	}

	values := map[string]string{
		"APP_NAME":      "myapp",
		"PORT":          "8080",
		"DEBUG":         "true",
		"RATE_LIMIT":    "0.5",
		"ALLOWED_HOSTS": "example.com, api.example.com",
	}
	requestedKeys := make([]string, 0, len(values))

	got, err := Parse[config](func(k string) (string, bool) {
		requestedKeys = append(requestedKeys, k)
		v, ok := values[k]
		return v, ok
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	wantHosts := []string{"example.com", "api.example.com"}
	if got.AppName != "myapp" || got.AppEnv != "production" || got.Port != 8080 || !got.Debug || got.RateLimit != 0.5 || got.Ignored != "" {
		t.Fatalf("Parse() = %+v", got)
	}
	if len(got.AllowedHosts) != len(wantHosts) || got.AllowedHosts[0] != wantHosts[0] || got.AllowedHosts[1] != wantHosts[1] {
		t.Fatalf("Parse().AllowedHosts = %+v, want %+v", got.AllowedHosts, wantHosts)
	}
	if strings.Join(requestedKeys, ",") != "APP_NAME,APP_ENV,PORT,DEBUG,RATE_LIMIT,ALLOWED_HOSTS" {
		t.Fatalf("requested keys = %+v", requestedKeys)
	}
}

func TestParseMissingRequiredField(t *testing.T) {
	type config struct {
		RequiredField string `env:"REQUIRED_FIELD"`
	}

	_, err := Parse[config](func(string) (string, bool) {
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), "env var REQUIRED_FIELD missing") {
		t.Fatalf("Parse() error = %v, want missing required field error", err)
	}
}

func TestParseUsesEmptyLoadedValue(t *testing.T) {
	type config struct {
		OptionalURL string `env:"OPTIONAL_URL,https://example.com"`
	}

	got, err := Parse[config](func(k string) (string, bool) {
		if k == "OPTIONAL_URL" {
			return "", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.OptionalURL != "" {
		t.Fatalf("Parse().OptionalURL = %q, want empty string", got.OptionalURL)
	}
}

func TestParsePointerFieldsAreOptional(t *testing.T) {
	type config struct {
		OptionalName *string `env:"OPTIONAL_NAME"`
		Port         *int    `env:"PORT"`
		Debug        *bool   `env:"DEBUG,true"`
	}

	got, err := Parse[config](func(k string) (string, bool) {
		if k == "PORT" {
			return "8080", true
		}
		return "", false
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.OptionalName != nil {
		t.Fatalf("Parse().OptionalName = %v, want nil", *got.OptionalName)
	}
	if got.Port == nil || *got.Port != 8080 {
		t.Fatalf("Parse().Port = %v, want pointer to 8080", got.Port)
	}
	if got.Debug == nil || *got.Debug != true {
		t.Fatalf("Parse().Debug = %v, want pointer to true", got.Debug)
	}
}

func TestParsePointerFieldDecodeError(t *testing.T) {
	type config struct {
		Port *int `env:"PORT"`
	}

	_, err := Parse[config](func(k string) (string, bool) {
		if k == "PORT" {
			return "not-a-number", true
		}
		return "", false
	})
	if err == nil || !strings.Contains(err.Error(), "failed to decode PORT as int") {
		t.Fatalf("Parse() error = %v, want int decode error", err)
	}
}

func TestMustParse(t *testing.T) {
	type config struct {
		AppName string `env:"APP_NAME"`
	}

	got := MustParse[config](func(k string) (string, bool) {
		if k == "APP_NAME" {
			return "myapp", true
		}
		return "", false
	})
	if got.AppName != "myapp" {
		t.Fatalf("MustParse().AppName = %q, want myapp", got.AppName)
	}
}
