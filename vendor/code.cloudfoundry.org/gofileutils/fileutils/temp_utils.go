package fileutils

import (
	"io/ioutil"
	"os"
)

func TempDir(namePrefix string, cb func(tmpDir string, err error)) {
	tmpDir, err := ioutil.TempDir("", namePrefix)

	defer func() {
		os.RemoveAll(tmpDir)
	}()

	cb(tmpDir, err)
}

func TempFile(namePrefix string, cb func(tmpFile *os.File, err error)) {
	tmpFile, err := ioutil.TempFile("", namePrefix)

	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	cb(tmpFile, err)
}
