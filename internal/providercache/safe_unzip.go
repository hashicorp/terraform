package providercache

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Copied verbatim from https://github.com/hashicorp/go-getter

// mode returns the file mode masked by the umask
func mode(mode, umask os.FileMode) os.FileMode {
	return mode & ^umask
}

// ZipDecompressor is an implementation of Decompressor that can
// decompress zip files.
type ZipDecompressor struct {
	// FileSizeLimit limits the total size of all
	// decompressed files.
	//
	// The zero value means no limit.
	FileSizeLimit int64

	// FilesLimit limits the number of files that are
	// allowed to be decompressed.
	//
	// The zero value means no limit.
	FilesLimit int
}

func (d *ZipDecompressor) Decompress(dst, src string, dir bool, umask os.FileMode) error {
	// If we're going into a directory we should make that first
	mkdir := dst
	if !dir {
		mkdir = filepath.Dir(dst)
	}
	if err := os.MkdirAll(mkdir, mode(0755, umask)); err != nil {
		return err
	}

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
	if !dir && len(zipR.File) > 1 {
		return fmt.Errorf("expected a single file: %s", src)
	}

	if d.FilesLimit > 0 && len(zipR.File) > d.FilesLimit {
		return fmt.Errorf("zip archive contains too many files: %d > %d", len(zipR.File), d.FilesLimit)
	}

	var fileSizeTotal int64

	// Go through and unarchive
	for _, f := range zipR.File {
		path := dst
		if dir {
			// Disallow parent traversal
			if containsDotDot(f.Name) {
				return fmt.Errorf("entry contains '..': %s", f.Name)
			}

			path = filepath.Join(path, f.Name)
		}

		fileInfo := f.FileInfo()

		fileSizeTotal += fileInfo.Size()

		if d.FileSizeLimit > 0 && fileSizeTotal > d.FileSizeLimit {
			return fmt.Errorf("zip archive larger than limit: %d", d.FileSizeLimit)
		}

		if fileInfo.IsDir() {
			if !dir {
				return fmt.Errorf("expected a single file: %s", src)
			}

			// A directory, just make the directory and continue unarchiving...
			if err := os.MkdirAll(path, mode(0755, umask)); err != nil {
				return err
			}

			continue
		}

		// Create the enclosing directories if we must. ZIP files aren't
		// required to contain entries for just the directories so this
		// can happen.
		if dir {
			if err := os.MkdirAll(filepath.Dir(path), mode(0755, umask)); err != nil {
				return err
			}
		}

		// Open the file for reading
		srcF, err := f.Open()
		if err != nil {
			if srcF != nil {
				srcF.Close()
			}
			return err
		}

		// Size limit is tracked using the returned file info.
		err = copyReader(path, srcF, f.Mode(), umask, 0)
		srcF.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// copyReader copies from an io.Reader into a file, using umask to create the dst file
// copied from https://github.com/hashicorp/go-getter get_file_copy.go
func copyReader(dst string, src io.Reader, fmode, umask os.FileMode, fileSizeLimit int64) error {
	dstF, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fmode)
	if err != nil {
		return err
	}
	defer dstF.Close()

	if fileSizeLimit > 0 {
		src = io.LimitReader(src, fileSizeLimit)
	}

	_, err = io.Copy(dstF, src)
	if err != nil {
		return err
	}

	// Explicitly chmod; the process umask is unconditionally applied otherwise.
	// We'll mask the mode with our own umask, but that may be different than
	// the process umask
	return os.Chmod(dst, mode(fmode, umask))
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
