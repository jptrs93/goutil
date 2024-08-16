package fileutil

import (
	"fmt"
	"os"
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

func MustEnsureDir(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(fmt.Sprintf("failed creating dir %v: %v", path, err))
	}
}
