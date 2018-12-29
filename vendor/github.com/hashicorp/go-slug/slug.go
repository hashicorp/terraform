package slug

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Meta provides detailed information about a slug.
type Meta struct {
	// The list of files contained in the slug.
	Files []string

	// Total size of the slug in bytes.
	Size int64
}

// Pack creates a slug from a directory src, and writes the new
// slug to w. Returns metadata about the slug and any error.
func Pack(src string, w io.Writer) (*Meta, error) {
	// Gzip compress all the output data
	gzipW := gzip.NewWriter(w)

	// Tar the file contents
	tarW := tar.NewWriter(gzipW)

	// Track the metadata details as we go.
	meta := &Meta{}

	// Walk the tree of files
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check the file type and if we need to write the body
		keepFile, writeBody := checkFileMode(info.Mode())
		if !keepFile {
			return nil
		}

		// Get the relative path from the unpack directory
		subpath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("Failed to get relative path for file %q: %v", path, err)
		}
		if subpath == "." {
			return nil
		}

		// Read the symlink target. We don't track the error because
		// it doesn't matter if there is an error.
		target, _ := os.Readlink(path)

		// Build the file header for the tar entry
		header, err := tar.FileInfoHeader(info, target)
		if err != nil {
			return fmt.Errorf("Failed creating archive header for file %q: %v", path, err)
		}

		// Modify the header to properly be the full subpath
		header.Name = subpath
		if info.IsDir() {
			header.Name += "/"
		}

		// Write the header first to the archive.
		if err := tarW.WriteHeader(header); err != nil {
			return fmt.Errorf("Failed writing archive header for file %q: %v", path, err)
		}

		// Account for the file in the list
		meta.Files = append(meta.Files, header.Name)

		// Skip writing file data for certain file types (above).
		if !writeBody {
			return nil
		}

		// Add the size since we are going to write the body.
		meta.Size += info.Size()

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Failed opening file %q for archiving: %v", path, err)
		}
		defer f.Close()

		if _, err = io.Copy(tarW, f); err != nil {
			return fmt.Errorf("Failed copying file %q to archive: %v", path, err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Flush the tar writer
	if err := tarW.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close the tar archive: %v", err)
	}

	// Flush the gzip writer
	if err := gzipW.Close(); err != nil {
		return nil, fmt.Errorf("Failed to close the gzip writer: %v", err)
	}

	return meta, nil
}

// Unpack is used to read and extract the contents of a slug to
// directory dst. Returns any error.
func Unpack(r io.Reader, dst string) error {
	// Decompress as we read
	uncompressed, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("Failed to uncompress slug: %v", err)
	}

	// Untar as we read
	untar := tar.NewReader(uncompressed)

	// Unpackage all the contents into the directory
	for {
		header, err := untar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Failed to untar slug: %v", err)
		}

		// Get rid of absolute paths
		path := header.Name
		if path[0] == '/' {
			path = path[1:]
		}
		path = filepath.Join(dst, path)

		// Make the directories to the path
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

		// Only unpack regular files from this point on
		if header.Typeflag == tar.TypeDir {
			continue
		} else if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("Failed creating %q: unsupported type %c", path,
				header.Typeflag)
		}

		// Open a handle to the destination
		fh, err := os.Create(path)
		if err != nil {
			// This mimics tar's behavior wrt the tar file containing duplicate files
			// and it allowing later ones to clobber earlier ones even if the file
			// has perms that don't allow overwriting
			if os.IsPermission(err) {
				os.Chmod(path, 0600)
				fh, err = os.Create(path)
			}

			if err != nil {
				return fmt.Errorf("Failed creating file %q: %v", path, err)
			}
		}

		// Copy the contents
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
	case m.IsRegular():
		return true, true

	case m.IsDir():
		return true, false

	case m&os.ModeSymlink != 0:
		return true, false
	}

	return false, false
}
