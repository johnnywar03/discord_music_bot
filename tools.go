package main

import (
	"os"
	"path/filepath"
)

func remove(filePath string) (err error) {
	directory, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer directory.Close()
	files, err := directory.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(filePath, file))
		if err != nil {
			return err
		}
	}
	return nil
}
