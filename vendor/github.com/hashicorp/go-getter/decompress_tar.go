package getter

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// untar is a shared helper for untarring an archive. The reader should provide
// an uncompressed view of the tar archive.
func untar(input io.Reader, dst, src string, dir bool) error {
	tarR := tar.NewReader(input)
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
			if !dir {
				return fmt.Errorf("expected a single file: %s", src)
			}

			// A directory, just make the directory and continue unarchiving...
			if err := os.MkdirAll(path, 0755); err != nil {
				return err
			}

			continue
		} else {
			// There is no ordering guarantee that a file in a directory is
			// listed before the directory
			dstPath := filepath.Dir(path)

			// Check that the directory exists, otherwise create it
			if _, err := os.Stat(dstPath); os.IsNotExist(err) {
				if err := os.MkdirAll(dstPath, 0755); err != nil {
					return err
				}
			}
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
