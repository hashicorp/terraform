package internal

import (
	"io"
	"io/fs"
	"os"
	"strings"
)

func cp(sourceDirectory, targetDirectory string, skipFiles []string) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		for _, skip := range skipFiles {
			if skip == entry.Name() {
				return nil
			}
		}

		if path == sourceDirectory {
			return nil
		}

		targetFile := strings.ReplaceAll(path, sourceDirectory, targetDirectory)

		if entry.IsDir() {
			return os.MkdirAll(targetFile, os.ModePerm)
		}

		source, err := os.Open(path)
		if err != nil {
			return err
		}
		defer source.Close()

		target, err := os.Create(targetFile)
		if err != nil {
			return err
		}
		defer target.Close()

		_, err = io.Copy(target, source)
		return err
	}
}
