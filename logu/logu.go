package logu

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func MustResolveLogDir(appName string) string {
	if dir, err := ResolveLogDir(appName); err != nil {
		panic(err)
	} else {
		return dir
	}
}

// ResolveLogDir returns an OS-appropriate default directory for application logs,
// then appends appName and ensures that final directory exists.
//
// Default base directories by platform:
//   - macOS (darwin): ~/Library/Logs
//   - Linux:
//   - $XDG_DATA_HOME when set
//   - otherwise ~/.local/share
//
// The returned path is always: <baseDir>/<appName>
//
// Examples:
//   - macOS + appName="myapp" -> ~/Library/Logs/myapp
//   - Linux + XDG_DATA_HOME=/home/alice/.local/state -> /home/alice/.local/state/myapp
//   - Linux without XDG_DATA_HOME -> ~/.local/share/myapp
//
// Unsupported operating systems return an error.
func ResolveLogDir(appName string) (string, error) {
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
	logDir := filepath.Join(baseDir, appName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}
	return logDir, nil
}
