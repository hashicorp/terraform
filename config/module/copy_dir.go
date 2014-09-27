package module

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// copyDir copies the src directory contents into dst. Both directories
// should already exist.
func copyDir(dst, src string) error {
	src, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == src {
			return nil
		}

		basePath := filepath.Base(path)
		if strings.HasPrefix(basePath, ".") {
			// Skip any dot files
			return nil
		}

		dstPath := filepath.Join(dst, basePath)

		// If we have a directory, make that subdirectory, then continue
		// the walk.
		if info.IsDir() {
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}

			return copyDir(dstPath, path)
		}

		// If we have a file, copy the contents.
		srcF, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcF.Close()

		dstF, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstF.Close()

		if _, err := io.Copy(dstF, srcF); err != nil {
			return err
		}

		// Chmod it
		return os.Chmod(dstPath, info.Mode())
	}

	return filepath.Walk(src, walkFn)
}
