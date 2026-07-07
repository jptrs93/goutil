package envu

import (
	"log/slog"
	"testing"
)

func TestDecodeSlogLevel(t *testing.T) {
	tests := []struct {
		value string
		want  slog.Level
	}{
		{value: "DEBUG", want: slog.LevelDebug},
		{value: "info", want: slog.LevelInfo},
		{value: "WARN", want: slog.LevelWarn},
		{value: "ERROR", want: slog.LevelError},
		{value: "WARN+4", want: slog.LevelWarn + 4},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, err := Decode[slog.Level](tt.value)
			if err != nil {
				t.Fatalf("Decode[slog.Level](%q) error = %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("Decode[slog.Level](%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestMustGetSlogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "WARN")

	got := MustGet[slog.Level]("LOG_LEVEL")
	if got != slog.LevelWarn {
		t.Fatalf("MustGet[slog.Level]() = %v, want %v", got, slog.LevelWarn)
	}
}

func TestMustGetOrDefaultSlogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "ERROR")

	got := MustGetOrDefault("LOG_LEVEL", slog.LevelInfo)
	if got != slog.LevelError {
		t.Fatalf("MustGetOrDefault[slog.Level]() = %v, want %v", got, slog.LevelError)
	}
}

func TestMustGetOrDefaultSlogLevelDefault(t *testing.T) {
	got := MustGetOrDefault("LOG_LEVEL", slog.LevelInfo)
	if got != slog.LevelInfo {
		t.Fatalf("MustGetOrDefault[slog.Level]() = %v, want %v", got, slog.LevelInfo)
	}
}
