package pathu

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// ResolveAppDataDir returns an OS-appropriate default directory for application data,
// then appends appName.
//
//   - macOS + appName="myapp" -> ~/Library/Application Support/myapp
//   - Linux + XDG_DATA_HOME=/home/alice/.local/share -> /home/alice/.local/share/myapp
//   - Linux without XDG_DATA_HOME -> ~/.local/share/myapp
//   - Linux system + appName="myapp" -> /var/lib/myapp
//
// Unsupported operating systems return an error.
func ResolveAppDataDir(appName string, isSystemUser bool) (string, error) {
	var baseDir string
	var err error
	switch runtime.GOOS {
	case "darwin":
		if isSystemUser {
			return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}
		baseDir, err = resolveDarwinAppDataHome()
	case "linux":
		if isSystemUser {
			baseDir = filepath.Join("/var", "lib")
		} else {
			baseDir, err = resolveXDGDataHome()
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, appName), nil
}

// ResolveAppLogDir returns an OS-appropriate default directory for application logs,
// then appends appName.
//
//   - macOS + appName="myapp" -> ~/Library/Logs/myapp
//   - Linux + XDG_STATE_HOME=/home/alice/.local/state -> /home/alice/.local/state/myapp
//   - Linux + only XDG_DATA_HOME=/home/alice/.local/share -> /home/alice/.local/share/myapp
//   - Linux without either -> ~/.local/state/myapp
//   - Linux system + appName="myapp" -> /var/log/myapp
//
// Unsupported operating systems return an error.
func ResolveAppLogDir(appName string, isSystemUser bool) (string, error) {
	var baseDir string
	var err error
	switch runtime.GOOS {
	case "darwin":
		if isSystemUser {
			return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		}
		baseDir, err = resolveDarwinAppLogHome()
	case "linux":
		if isSystemUser {
			baseDir = filepath.Join("/var", "log")
		} else {
			baseDir, err = resolveXDGStateHome()
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, appName), nil
}
