package providercache

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// safeUnzip safely unzips a zip file named src into dstDir
// Based on https://github.com/hashicorp/go-getter ZipDecompressor.Decompress()
// Simplified, removed dead code due to how we call it.
// Changed to use atomic writer.
func safeUnzip(dstDir, src string) error {
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

		path := filepath.Join(dstDir, f.Name)

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

		err = writeAtomically(path, srcF, f.Mode())
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

// writeAtomically copies from an io.Reader into a file atomically.
// It guarantees that the file is complete and has the correct file
// mode before it becomes accessible at its final name, will handle
// multiple callers trying to create the same file simultaneously,
// and avoids ETXTBSY problems when overwriting running executables.
//
// Adapted from from https://github.com/hashicorp/go-getter get_file_copy.go copyReader()
func writeAtomically(dstName string, src io.Reader, fmode os.FileMode) error {
	dir, file := filepath.Split(dstName)
	if dir == "" {
		dir = "."
	}
	tmpFile, err := os.CreateTemp(dir, file)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %v", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		if err != nil {
			// Ignore errors in cleanup, just do the best we can
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = io.Copy(tmpFile, src); err != nil {
		return fmt.Errorf("cannot write to temp file: %v", err)
	}
	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("cannot flush tmp file: %v", err)
	}
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("cannot close tmp file: %v", err)
	}
	if err = os.Chmod(tmpName, fmode); err != nil {
		return fmt.Errorf("cannot set mode to 0%o: %v", fmode, err)
	}
	if err = os.Rename(tmpName, dstName); err != nil {
		// It's possible for different users to share the same cache directory,
		// and sometimes terraform downloads providers when it shouldn't.
		// As a last resort, compare new and old file, if they are identical
		// we are still good.
		contentIsIdentical, err2 := compareFileContent(tmpName, dstName)
		if err2 != nil {
			return err2
		}
		if !contentIsIdentical {
			return fmt.Errorf("cannot rename files, and content is different: %v", err)
		}
	}
	return nil
}

// Compare the contents of two files. If the contents are the same, return true.
// We very strongly suspect the files to be the same, so we just do a full
// content compare.
func compareFileContent(file1, file2 string) (bool, error) {
	f1, err := os.Open(file1)
	if err != nil {
		return false, err
	}
	defer f1.Close()
	f2, err := os.Open(file2)
	if err != nil {
		return false, err
	}
	defer f2.Close()

	buf1 := make([]byte, 4096)
	buf2 := make([]byte, 4096)
	for {
		len1, err1 := io.ReadFull(f1, buf1)
		len2, err2 := io.ReadFull(f2, buf2)
		if len1 != len2 {
			return false, nil
		}
		if !bytes.Equal(buf1[:len1], buf2[:len2]) {
			return false, nil
		}
		if (err1 == io.EOF && err2 == io.EOF) || (err1 == io.ErrUnexpectedEOF && err2 == io.ErrUnexpectedEOF) {
			return true, nil
		}
		if err1 != nil {
			return false, err1
		}
		if err2 != nil {
			return false, err2
		}
	}
}
