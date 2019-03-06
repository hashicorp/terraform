package initwd

import (
	"io"
	"io/ioutil"
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

		if strings.HasPrefix(filepath.Base(path), ".") {
			// Skip any dot files
			if info.IsDir() {
				return filepath.SkipDir
			} else {
				return nil
			}
		}

		// The "path" has the src prefixed to it. We need to join our
		// destination with the path without the src on it.
		dstPath := filepath.Join(dst, path[len(src):])

		// we don't want to try and copy the same file over itself.
		if eq, err := sameFile(path, dstPath); eq {
			return nil
		} else if err != nil {
			return err
		}

		// If we have a directory, make that subdirectory, then continue
		// the walk.
		if info.IsDir() {
			if path == filepath.Join(src, dst) {
				// dst is in src; don't walk it.
				return nil
			}

			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}

			return nil
		}

		// info.IsDir() returns false for symlinks, even if the symlink is a
		// directory.
		// TODO: This may be fixed - or different - in go1.12, at which point we
		// can remove this ugly, duplicate block
		if info.Mode()&os.ModeSymlink == os.ModeSymlink {
			_, err := ioutil.ReadDir(path)
			if err == nil {
				// it was a directory ALL ALONG!!!
				if path == filepath.Join(src, dst) {
					// dst is in src; don't walk it.
					return nil
				}

				if err := os.MkdirAll(dstPath, 0755); err != nil {
					return err
				}

				return nil
			}
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

// sameFile tried to determine if to paths are the same file.
// If the paths don't match, we lookup the inode on supported systems.
func sameFile(a, b string) (bool, error) {
	if a == b {
		return true, nil
	}

	aIno, err := inode(a)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	bIno, err := inode(b)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	if aIno > 0 && aIno == bIno {
		return true, nil
	}

	return false, nil
}
