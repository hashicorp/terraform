package providercache

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Based on https://github.com/hashicorp/go-getter
// Simplified, removed dead code

func unzip(dst, src string) error {
	// Open the zip
	zipR, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zipR.Close()

	// Check the zip integrity
	if len(zipR.File) == 0 {
		// Empty archive
		return fmt.Errorf("empty archive: %s", src)
	}

	// Go through and unarchive
	for _, f := range zipR.File {
		// Disallow parent traversal
		if containsDotDot(f.Name) {
			return fmt.Errorf("entry contains '..': %s", f.Name)
		}

		path := filepath.Join(dst, f.Name)

		if f.FileInfo().IsDir() {
			// A directory, just make the directory and continue unarchiving...
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}
			continue
		}

		// Create the enclosing directories if we must. ZIP files aren't
		// required to contain entries for just the directories so this
		// can happen.
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Open the file for reading
		srcF, err := f.Open()
		if err != nil {
			if srcF != nil {
				srcF.Close()
			}
			return err
		}

		err = copyReader(path, srcF, f.Mode())
		srcF.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// containsDotDot checks if the filepath value v contains a ".." entry.
// This will check filepath components by splitting along / or \. This
// function is copied directly from the Go net/http implementation.
func containsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

// copyReader copies from an io.Reader into a file, using umask to create the dst file
// copied from https://github.com/hashicorp/go-getter get_file_copy.go
func copyReader(dst string, src io.Reader, fmode os.FileMode) error {
	dstF, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fmode)
	if err != nil {
		return err
	}
	defer dstF.Close()

	_, err = io.Copy(dstF, src)
	if err != nil {
		return err
	}

	// Explicitly chmod; the process umask is unconditionally applied otherwise.
	// We'll mask the mode with our own umask, but that may be different than
	// the process umask
	return os.Chmod(dst, fmode)
}
