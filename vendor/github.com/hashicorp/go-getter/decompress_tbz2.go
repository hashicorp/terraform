package getter

import (
	"archive/tar"
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TarBzip2Decompressor is an implementation of Decompressor that can
// decompress tar.bz2 files.
type TarBzip2Decompressor struct{}

func (d *TarBzip2Decompressor) Decompress(dst, src string, dir bool) error {
	// If we're going into a directory we should make that first
	mkdir := dst
	if !dir {
		mkdir = filepath.Dir(dst)
	}
	if err := os.MkdirAll(mkdir, 0755); err != nil {
		return err
	}

	// File first
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	// Bzip2 compression is second
	bzipR := bzip2.NewReader(f)

	// Once bzip decompressed we have a tar format
	tarR := tar.NewReader(bzipR)
	done := false
	for {
		hdr, err := tarR.Next()
		if err == io.EOF {
			if !done {
				// Empty archive
				return fmt.Errorf("empty archive: %s", src)
			}

			return nil
		}
		if err != nil {
			return err
		}

		path := dst
		if dir {
			path = filepath.Join(path, hdr.Name)
		}

		if hdr.FileInfo().IsDir() {
			if dir {
				return fmt.Errorf("expected a single file: %s", src)
			}

			// A directory, just make the directory and continue unarchiving...
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}

			continue
		}

		// We have a file. If we already decoded, then it is an error
		if !dir && done {
			return fmt.Errorf("expected a single file, got multiple: %s", src)
		}

		// Mark that we're done so future in single file mode errors
		done = true

		// Open the file for writing
		dstF, err := os.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(dstF, tarR)
		dstF.Close()
		if err != nil {
			return err
		}

		// Chmod the file
		if err := os.Chmod(path, hdr.FileInfo().Mode()); err != nil {
			return err
		}
	}
}
