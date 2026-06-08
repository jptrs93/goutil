package pathu

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	got, err := resolveXDGDataHome()
	if err != nil {
		t.Fatalf("resolveXDGDataHome() error = %v", err)
	}
	if got != "/tmp/xdg-data" {
		t.Fatalf("resolveXDGDataHome() = %q, want /tmp/xdg-data", got)
	}
}

func TestResolveXDGDataHomeDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME fallback is Unix-specific")
	}

	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", home)

	got, err := resolveXDGDataHome()
	if err != nil {
		t.Fatalf("resolveXDGDataHome() error = %v", err)
	}
	want := filepath.Join(home, ".local", "share")
	if got != want {
		t.Fatalf("resolveXDGDataHome() = %q, want %q", got, want)
	}
}

func TestResolveXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	got, err := resolveXDGStateHome()
	if err != nil {
		t.Fatalf("resolveXDGStateHome() error = %v", err)
	}
	if got != "/tmp/xdg-state" {
		t.Fatalf("resolveXDGStateHome() = %q, want /tmp/xdg-state", got)
	}
}

func TestResolveXDGStateHomeFallsBackToDataHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg-data")

	got, err := resolveXDGStateHome()
	if err != nil {
		t.Fatalf("resolveXDGStateHome() error = %v", err)
	}
	if got != "/tmp/xdg-data" {
		t.Fatalf("resolveXDGStateHome() = %q, want /tmp/xdg-data", got)
	}
}

func TestResolveXDGStateHomeDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME fallback is Unix-specific")
	}

	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", home)

	got, err := resolveXDGStateHome()
	if err != nil {
		t.Fatalf("resolveXDGStateHome() error = %v", err)
	}
	want := filepath.Join(home, ".local", "state")
	if got != want {
		t.Fatalf("resolveXDGStateHome() = %q, want %q", got, want)
	}
}
