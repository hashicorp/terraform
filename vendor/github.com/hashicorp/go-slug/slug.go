package slug

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Meta provides detailed information about a slug.
type Meta struct {
	// The list of files contained in the slug.
	Files []string

	// Total size of the slug in bytes.
	Size int64
}

// Pack creates a slug from a src directory, and writes the new slug
// to w. Returns metadata about the slug and any errors.
//
// When dereference is set to true, symlinks with a target outside of
// the src directory will be dereferenced. When dereference is set to
// false symlinks with a target outside the src directory are omitted
// from the slug.
func Pack(src string, w io.Writer, dereference bool) (*Meta, error) {
	// Gzip compress all the output data.
	gzipW := gzip.NewWriter(w)

	// Tar the file contents.
	tarW := tar.NewWriter(gzipW)

	// Load the ignore rule configuration, which will use
	// defaults if no .terraformignore is configured
	ignoreRules := parseIgnoreFile(src)

	// Track the metadata details as we go.
	meta := &Meta{}

	// Walk the tree of files.
	err := filepath.Walk(src, packWalkFn(src, src, src, tarW, meta, dereference, ignoreRules))
	if err != nil {
		return nil, err
	}

	// Flush the tar writer.
	if err := tarW.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close the tar archive: %v", err)
	}

	// Flush the gzip writer.
	if err := gzipW.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close the gzip writer: %v", err)
	}

	return meta, nil
}

func packWalkFn(root, src, dst string, tarW *tar.Writer, meta *Meta, dereference bool, ignoreRules []rule) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path from the current src directory.
		subpath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for file %q: %v", path, err)
		}
		if subpath == "." {
			return nil
		}

		if m := matchIgnoreRule(subpath, ignoreRules); m {
			return nil
		}

		// Catch directories so we don't end up with empty directories,
		// the files are ignored correctly
		if info.IsDir() {
			if m := matchIgnoreRule(subpath+string(os.PathSeparator), ignoreRules); m {
				return nil
			}
		}

		// Get the relative path from the initial root directory.
		subpath, err = filepath.Rel(root, strings.Replace(path, src, dst, 1))
		if err != nil {
			return fmt.Errorf("Failed to get relative path for file %q: %v", path, err)
		}
		if subpath == "." {
			return nil
		}

		// Check the file type and if we need to write the body.
		keepFile, writeBody := checkFileMode(info.Mode())
		if !keepFile {
			return nil
		}

		fm := info.Mode()
		header := &tar.Header{
			Name:    filepath.ToSlash(subpath),
			ModTime: info.ModTime(),
			Mode:    int64(fm.Perm()),
		}

		switch {
		case info.IsDir():
			header.Typeflag = tar.TypeDir
			header.Name += "/"

		case fm.IsRegular():
			header.Typeflag = tar.TypeReg
			header.Size = info.Size()

		case fm&os.ModeSymlink != 0:
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("Failed to get symbolic link destination for %q: %v", path, err)
			}

			// If the target is within the current source, we
			// create the symlink using a relative path.
			if strings.Contains(target, src) {
				link, err := filepath.Rel(filepath.Dir(path), target)
				if err != nil {
					return fmt.Errorf("Failed to get relative path for symlink destination %q: %v", target, err)
				}

				header.Typeflag = tar.TypeSymlink
				header.Linkname = filepath.ToSlash(link)

				// Break out of the case as a symlink
				// doesn't need any additional config.
				break
			}

			if !dereference {
				// Return early as the symlink has a target outside of the
				// src directory and we don't want to dereference symlinks.
				return nil
			}

			// Get the file info for the target.
			info, err = os.Lstat(target)
			if err != nil {
				return fmt.Errorf("Failed to get file info from file %q: %v", target, err)
			}

			// If the target is a directory we can recurse into the target
			// directory by calling the packWalkFn with updated arguments.
			if info.IsDir() {
				return filepath.Walk(target, packWalkFn(root, target, path, tarW, meta, dereference, ignoreRules))
			}

			// Dereference this symlink by updating the header with the target file
			// details and set writeBody to true so the body will be written.
			header.Typeflag = tar.TypeReg
			header.ModTime = info.ModTime()
			header.Mode = int64(info.Mode().Perm())
			header.Size = info.Size()
			writeBody = true

		default:
			return fmt.Errorf("Unexpected file mode %v", fm)
		}

		// Write the header first to the archive.
		if err := tarW.WriteHeader(header); err != nil {
			return fmt.Errorf("Failed writing archive header for file %q: %v", path, err)
		}

		// Account for the file in the list.
		meta.Files = append(meta.Files, header.Name)

		// Skip writing file data for certain file types (above).
		if !writeBody {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Failed opening file %q for archiving: %v", path, err)
		}
		defer f.Close()

		size, err := io.Copy(tarW, f)
		if err != nil {
			return fmt.Errorf("Failed copying file %q to archive: %v", path, err)
		}

		// Add the size we copied to the body.
		meta.Size += size

		return nil
	}
}

// Unpack is used to read and extract the contents of a slug to
// the dst directory. Returns any errors.
func Unpack(r io.Reader, dst string) error {
	// Decompress as we read.
	uncompressed, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("Failed to uncompress slug: %v", err)
	}

	// Untar as we read.
	untar := tar.NewReader(uncompressed)

	// Unpackage all the contents into the directory.
	for {
		header, err := untar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Failed to untar slug: %v", err)
		}

		// Get rid of absolute paths.
		path := header.Name
		if path[0] == '/' {
			path = path[1:]
		}
		path = filepath.Join(dst, path)

		// Make the directories to the path.
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("Failed to create directory %q: %v", dir, err)
		}

		// If we have a symlink, just link it.
		if header.Typeflag == tar.TypeSymlink {
			if err := os.Symlink(header.Linkname, path); err != nil {
				return fmt.Errorf("Failed creating symlink %q => %q: %v",
					path, header.Linkname, err)
			}
			continue
		}

		// Only unpack regular files from this point on.
		if header.Typeflag == tar.TypeDir {
			continue
		} else if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("Failed creating %q: unsupported type %c", path,
				header.Typeflag)
		}

		// Open a handle to the destination.
		fh, err := os.Create(path)
		if err != nil {
			// This mimics tar's behavior wrt the tar file containing duplicate files
			// and it allowing later ones to clobber earlier ones even if the file
			// has perms that don't allow overwriting.
			if os.IsPermission(err) {
				os.Chmod(path, 0600)
				fh, err = os.Create(path)
			}

			if err != nil {
				return fmt.Errorf("Failed creating file %q: %v", path, err)
			}
		}

		// Copy the contents.
		_, err = io.Copy(fh, untar)
		fh.Close()
		if err != nil {
			return fmt.Errorf("Failed to copy slug file %q: %v", path, err)
		}

		// Restore the file mode. We have to do this after writing the file,
		// since it is possible we have a read-only mode.
		mode := header.FileInfo().Mode()
		if err := os.Chmod(path, mode); err != nil {
			return fmt.Errorf("Failed setting permissions on %q: %v", path, err)
		}
	}
	return nil
}

// checkFileMode is used to examine an os.FileMode and determine if it should
// be included in the archive, and if it has a data body which needs writing.
func checkFileMode(m os.FileMode) (keep, body bool) {
	switch {
	case m.IsDir():
		return true, false

	case m.IsRegular():
		return true, true

	case m&os.ModeSymlink != 0:
		return true, false
	}

	return false, false
}
