package envu

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var loadDotEnvOnce = sync.Once{}

func LoadDotEnv(missingOk bool) {
	loadDotEnvOnce.Do(func() {
		path, err := resolveDotEnvFile()
		if err != nil && !missingOk {
			panic(err)
		}
		if path == "" {
			fmt.Println("no .env file found")
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("failed to read env file %s: %v", path, err))
		}
		env, err := ParseDotEnvBytes(data)
		if err != nil {
			panic(fmt.Sprintf("failed to parse env file %s: %v", path, err))
		}
		for key, value := range env {
			// don't overload existing variables
			if _, exists := os.LookupEnv(key); !exists {
				if err := os.Setenv(key, value); err != nil {
					panic(fmt.Sprintf("failed to set environment variable %s: %v", key, err))
				}
			}
		}
	})
}

func SearchForFile(searchDepth int, searchNames ...string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %v", err)
	}
	dir := cwd
	depth := 0
	for {
		for _, name := range searchNames {
			envPath := filepath.Join(dir, name)
			if _, err = os.Stat(envPath); err == nil {
				return envPath, nil
			}
		}
		depth += 1
		parent := filepath.Dir(dir)
		if depth >= searchDepth || parent == dir {
			return "", fmt.Errorf("no .env file found in current directory or ancestors to depth: %v", searchDepth)
		}
		dir = parent
	}
}

func resolveDotEnvFile() (string, error) {
	f, ok := os.LookupEnv("DOT_ENV_FILE")
	if ok && f != "" {
		return f, nil
	}
	return SearchForFile(10, ".env")
}
