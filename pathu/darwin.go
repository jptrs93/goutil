package pathu

import (
	"fmt"
	"os"
	"path/filepath"
)

func resolveDarwinAppDataHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}

func resolveDarwinAppLogHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, "Library", "Logs"), nil
}
