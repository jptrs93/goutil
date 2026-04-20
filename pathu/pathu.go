package pathu

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveAppDataDir returns an OS-appropriate default directory for application data,
// then appends appName and ensures that final directory exists.
//
// Default base directories by platform:
//   - macOS (darwin): ~/Library/Application Support
//   - Linux:
//   - $XDG_DATA_HOME when set
//   - otherwise ~/.local/share
//
// The returned path is always: <baseDir>/<appName>
//
// Examples:
//   - macOS + appName="myapp" -> ~/Library/Application Support/myapp
//   - Linux + XDG_DATA_HOME=/home/alice/.local/share -> /home/alice/.local/share/myapp
//   - Linux without XDG_DATA_HOME -> ~/.local/share/myapp
//
// Unsupported operating systems return an error.
func ResolveAppDataDir(appName string) (string, error) {
	var baseDir string
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(home, "Library", "Application Support")
	case "linux":
		// Linux: Follow XDG Base Directory specification
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			baseDir = xdgDataHome
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			baseDir = filepath.Join(home, ".local", "share")
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	dataDir := filepath.Join(baseDir, appName)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}
	return dataDir, nil
}

// ResolveAppLogDir returns an OS-appropriate default directory for application logs,
// then appends appName and ensures that final directory exists.
//
// Default base directories by platform:
//   - macOS (darwin): ~/Library/Logs
//   - Linux (in priority order):
//   - $XDG_STATE_HOME when set
//   - $XDG_DATA_HOME when set
//   - otherwise ~/.local/state
//
// The returned path is always: <baseDir>/<appName>
//
// Examples:
//   - macOS + appName="myapp" -> ~/Library/Logs/myapp
//   - Linux + XDG_STATE_HOME=/home/alice/.local/state -> /home/alice/.local/state/myapp
//   - Linux + only XDG_DATA_HOME=/home/alice/.local/share -> /home/alice/.local/share/myapp
//   - Linux without either -> ~/.local/state/myapp
//
// Unsupported operating systems return an error.
func ResolveAppLogDir(appName string) (string, error) {
	var baseDir string
	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Logs
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = filepath.Join(home, "Library", "Logs")
	case "linux":
		// Linux: Follow XDG Base Directory specification.
		// Prefer XDG_STATE_HOME (intended for logs), fall back to XDG_DATA_HOME,
		// then to the default ~/.local/state.
		if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
			baseDir = xdgStateHome
		} else if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			baseDir = xdgDataHome
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			baseDir = filepath.Join(home, ".local", "state")
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	logDir := filepath.Join(baseDir, appName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}
	return logDir, nil
}
