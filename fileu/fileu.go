package fileu

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func AppendTo(fileName string, data []byte) error {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

func EnsureDir(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func EnsureDirWithPerm(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("creating dir %q: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod dir %q: %w", path, err)
	}
	return nil
}

func MustEnsureDir(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(fmt.Sprintf("failed creating dir %v: %v", path, err))
	}
}

func RemoveFilesWithPrefix(dir string, filePrefix string) error {

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasPrefix(info.Name(), filePrefix) {
			err := os.Remove(path)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %w", path, err)
			}
		}

		return nil
	})

	return err
}
