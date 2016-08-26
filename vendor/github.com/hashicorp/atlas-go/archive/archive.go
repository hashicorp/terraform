// archive is package that helps create archives in a format that
// Atlas expects with its various upload endpoints.
package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Archive is the resulting archive. The archive data is generally streamed
// so the io.ReadCloser can be used to backpressure the archive progress
// and avoid memory pressure.
type Archive struct {
	io.ReadCloser

	Size     int64
	Metadata map[string]string
}

// ArchiveOpts are the options for defining how the archive will be built.
type ArchiveOpts struct {
	// Exclude and Include are filters of files to include/exclude in
	// the archive when creating it from a directory. These filters should
	// be relative to the packaging directory and should be basic glob
	// patterns.
	Exclude []string
	Include []string

	// Extra is a mapping of extra files to include within the archive. The
	// key should be the path within the archive and the value should be
	// an absolute path to the file to put into the archive. These extra
	// files will override any other files in the archive.
	Extra map[string]string

	// VCS, if true, will detect and use a VCS system to determine what
	// files to include the archive.
	VCS bool
}

// IsSet says whether any options were set.
func (o *ArchiveOpts) IsSet() bool {
	return len(o.Exclude) > 0 || len(o.Include) > 0 || o.VCS
}

// Constants related to setting special values for Extra in ArchiveOpts.
const (
	// ExtraEntryDir just creates the Extra key as a directory entry.
	ExtraEntryDir = ""
)

// CreateArchive takes the given path and ArchiveOpts and archives it.
//
// The archive will be fully completed and put into a temporary file.
// This must be done to retrieve the content length of the archive which
// is needed for almost all operations involving archives with Atlas. Because
// of this, sufficient disk space will be required to buffer the archive.
func CreateArchive(path string, opts *ArchiveOpts) (*Archive, error) {
	log.Printf("[INFO] creating archive from %s", path)

	// Dereference any symlinks and determine the real path and info
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		path, fi, err = readLinkFull(path, fi)
		if err != nil {
			return nil, err
		}
	}

	// Windows
	path = filepath.ToSlash(path)

	// Direct file paths cannot have archive options
	if !fi.IsDir() && opts.IsSet() {
		return nil, fmt.Errorf(
			"options such as exclude, include, and VCS can't be set when " +
				"the path is a file.")
	}

	if fi.IsDir() {
		return archiveDir(path, opts)
	} else {
		return archiveFile(path)
	}
}

func archiveFile(path string) (*Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if _, err := gzip.NewReader(f); err == nil {
		// Reset the read offset for future reading
		if _, err := f.Seek(0, 0); err != nil {
			f.Close()
			return nil, err
		}

		// Get the file info for the size
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, err
		}

		// This is a gzip file, let it through.
		return &Archive{ReadCloser: f, Size: fi.Size()}, nil
	}

	// Close the file, no use for it anymore
	f.Close()

	// We have a single file that is not gzipped. Compress it.
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	// Act like we're compressing a directory, but only include this one
	// file.
	return archiveDir(filepath.Dir(path), &ArchiveOpts{
		Include: []string{filepath.Base(path)},
	})
}

func archiveDir(root string, opts *ArchiveOpts) (*Archive, error) {

	var vcsInclude []string
	var metadata map[string]string
	if opts.VCS {
		var err error

		if err = vcsPreflight(root); err != nil {
			return nil, err
		}

		vcsInclude, err = vcsFiles(root)
		if err != nil {
			return nil, err
		}

		metadata, err = vcsMetadata(root)
		if err != nil {
			return nil, err
		}
	}

	// Make sure the root path is absolute
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	// Create the temporary file that we'll send the archive data to.
	archiveF, err := ioutil.TempFile("", "atlas-archive")
	if err != nil {
		return nil, err
	}

	// Create the wrapper for the result which will automatically
	// remove the temporary file on close.
	archiveWrapper := &readCloseRemover{F: archiveF}

	// Buffer the writer so that we can push as much data to disk at
	// a time as possible. 4M should be good.
	bufW := bufio.NewWriterSize(archiveF, 4096*1024)

	// Gzip compress all the output data
	gzipW := gzip.NewWriter(bufW)

	// Tar the file contents
	tarW := tar.NewWriter(gzipW)

	// First, walk the path and do the normal files
	werr := filepath.Walk(root, copyDirWalkFn(
		tarW, root, "", opts, vcsInclude))
	if werr == nil {
		// If that succeeded, handle the extra files
		werr = copyExtras(tarW, opts.Extra)
	}

	// Attempt to close all the things. If we get an error on the way
	// and we haven't had an error yet, then record that as the critical
	// error. But we still try to close everything.

	// Close the tar writer
	if err := tarW.Close(); err != nil && werr == nil {
		werr = err
	}

	// Close the gzip writer
	if err := gzipW.Close(); err != nil && werr == nil {
		werr = err
	}

	// Flush the buffer
	if err := bufW.Flush(); err != nil && werr == nil {
		werr = err
	}

	// If we had an error, then close the file (removing it) and
	// return the error.
	if werr != nil {
		archiveWrapper.Close()
		return nil, werr
	}

	// Seek to the beginning
	if _, err := archiveWrapper.F.Seek(0, 0); err != nil {
		archiveWrapper.Close()
		return nil, err
	}

	// Get the file information so we can get the size
	fi, err := archiveWrapper.F.Stat()
	if err != nil {
		archiveWrapper.Close()
		return nil, err
	}

	return &Archive{
		ReadCloser: archiveWrapper,
		Size:       fi.Size(),
		Metadata:   metadata,
	}, nil
}

func copyDirWalkFn(
	tarW *tar.Writer, root string, prefix string,
	opts *ArchiveOpts, vcsInclude []string) filepath.WalkFunc {

	errFunc := func(err error) filepath.WalkFunc {
		return func(string, os.FileInfo, error) error {
			return err
		}
	}

	// Windows
	root = filepath.ToSlash(root)

	var includeMap map[string]struct{}

	// If we have an include/exclude pattern set, then setup the lookup
	// table to determine what we want to include.
	if opts != nil && len(opts.Include) > 0 {
		includeMap = make(map[string]struct{})
		for _, pattern := range opts.Include {
			matches, err := filepath.Glob(filepath.Join(root, pattern))
			if err != nil {
				return errFunc(fmt.Errorf(
					"error checking include glob '%s': %s",
					pattern, err))
			}

			for _, path := range matches {
				// Windows
				path = filepath.ToSlash(path)
				subpath, err := filepath.Rel(root, path)
				subpath = filepath.ToSlash(subpath)

				if err != nil {
					return errFunc(err)
				}

				for {
					includeMap[subpath] = struct{}{}
					subpath = filepath.Dir(subpath)
					if subpath == "." {
						break
					}
				}
			}
		}
	}

	return func(path string, info os.FileInfo, err error) error {
		path = filepath.ToSlash(path)

		if err != nil {
			return err
		}

		// Get the relative path from the path since it contains the root
		// plus the path.
		subpath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if subpath == "." {
			return nil
		}
		if prefix != "" {
			subpath = filepath.Join(prefix, subpath)
		}
		// Windows
		subpath = filepath.ToSlash(subpath)

		// If we have a list of VCS files, check that first
		skip := false
		if len(vcsInclude) > 0 {
			skip = true
			for _, f := range vcsInclude {
				if f == subpath {
					skip = false
					break
				}

				if info.IsDir() && strings.HasPrefix(f, subpath+"/") {
					skip = false
					break
				}
			}
		}

		// If include is present, we only include what is listed
		if len(includeMap) > 0 {
			if _, ok := includeMap[subpath]; !ok {
				skip = true
			}
		}

		// If exclude, it is one last gate to excluding files
		if opts != nil {
			for _, exclude := range opts.Exclude {
				match, err := filepath.Match(exclude, subpath)
				if err != nil {
					return err
				}
				if match {
					skip = true
					break
				}
			}
		}

		// If we have to skip this file, then skip it, properly skipping
		// children if we're a directory.
		if skip {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		// If this is a symlink, then we need to get the symlink target
		// rather than the symlink itself.
		if info.Mode()&os.ModeSymlink != 0 {
			target, info, err := readLinkFull(path, info)
			if err != nil {
				return err
			}

			// Copy the concrete entry for this path. This will either
			// be the file itself or just a directory entry.
			if err := copyConcreteEntry(tarW, subpath, target, info); err != nil {
				return err
			}

			if info.IsDir() {
				return filepath.Walk(target, copyDirWalkFn(
					tarW, target, subpath, opts, vcsInclude))
			}
		}

		return copyConcreteEntry(tarW, subpath, path, info)
	}
}

func copyConcreteEntry(
	tarW *tar.Writer, entry string,
	path string, info os.FileInfo) error {
	// Windows
	path = filepath.ToSlash(path)

	// Build the file header for the tar entry
	header, err := tar.FileInfoHeader(info, path)
	if err != nil {
		return fmt.Errorf(
			"failed creating archive header: %s", path)
	}

	// Modify the header to properly be the full entry name
	header.Name = entry
	if info.IsDir() {
		header.Name += "/"
	}

	// Write the header first to the archive.
	if err := tarW.WriteHeader(header); err != nil {
		return fmt.Errorf(
			"failed writing archive header: %s", path)
	}

	// If it is a directory, then we're done (no body to write)
	if info.IsDir() {
		return nil
	}

	// Open the real file to write the data
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf(
			"failed opening file '%s' to write compressed archive.", path)
	}
	defer f.Close()

	if _, err = io.Copy(tarW, f); err != nil {
		return fmt.Errorf(
			"failed copying file to archive: %s", path)
	}

	return nil
}

func copyExtras(w *tar.Writer, extra map[string]string) error {
	var tmpDir string
	defer func() {
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
	}()

	for entry, path := range extra {
		// If the path is empty, then we set it to a generic empty directory
		if path == "" {
			// If tmpDir is still empty, then we create an empty dir
			if tmpDir == "" {
				td, err := ioutil.TempDir("", "archive")
				if err != nil {
					return err
				}

				tmpDir = td
			}

			path = tmpDir
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		// No matter what, write the entry. If this is a directory,
		// it'll just write the directory header.
		if err := copyConcreteEntry(w, entry, path, info); err != nil {
			return err
		}

		// If this is a directory, then we walk the internal contents
		// and copy those as well.
		if info.IsDir() {
			err := filepath.Walk(path, copyDirWalkFn(
				w, path, entry, nil, nil))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func readLinkFull(path string, info os.FileInfo) (string, os.FileInfo, error) {
	// Read the symlink continously until we reach a concrete file.
	target := path
	tries := 0
	for info.Mode()&os.ModeSymlink != 0 {
		var err error
		target, err = os.Readlink(target)
		if err != nil {
			return "", nil, err
		}
		if !filepath.IsAbs(target) {
			target, err = filepath.Abs(target)
			if err != nil {
				return "", nil, err
			}
		}
		info, err = os.Lstat(target)
		if err != nil {
			return "", nil, err
		}

		tries++
		if tries > 100 {
			return "", nil, fmt.Errorf(
				"Symlink for %s is too deep, over 100 levels deep",
				path)
		}
	}

	return target, info, nil
}

// readCloseRemover is an io.ReadCloser implementation that will remove
// the file on Close(). We use this to clean up our temporary file for
// the archive.
type readCloseRemover struct {
	F *os.File
}

func (r *readCloseRemover) Read(p []byte) (int, error) {
	return r.F.Read(p)
}

func (r *readCloseRemover) Close() error {
	// First close the file
	err := r.F.Close()

	// Next make sure to remove it, or at least try, regardless of error
	// above.
	os.Remove(r.F.Name())

	return err
}
