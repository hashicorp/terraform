// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package replacefile

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// AtomicWriteFile uses a temporary file along with this package's AtomicRename
// function in order to provide a replacement for ioutil.WriteFile that
// writes the given file into place as atomically as the underlying operating
// system can support.
//
// The sense of "atomic" meant by this function is that the file at the
// given filename will either contain the entirety of the previous contents
// or the entirety of the given data array if opened and read at any point
// during the execution of the function.
//
// On some platforms attempting to overwrite a file that has at least one
// open filehandle will produce an error. On other platforms, the overwriting
// will succeed but existing open handles will still refer to the old file,
// even though its directory entry is no longer present.
//
// Although AtomicWriteFile tries its best to avoid leaving behind its
// temporary file on error, some particularly messy error cases may result
// in a leftover temporary file.
func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	dir, file := filepath.Split(filename)
	if dir == "" {
		// If the file is in the current working directory then dir will
		// end up being "", but that's not right here because TempFile
		// treats an empty dir as meaning "use the TMPDIR environment variable".
		dir = "."
	}
	f, err := ioutil.TempFile(dir, file) // alongside target file and with a similar name
	if err != nil {
		return fmt.Errorf("cannot create temporary file to update %s: %s", filename, err)
	}
	tmpName := f.Name()
	moved := false
	defer func(f *os.File, name string) {
		// Remove the temporary file if it hasn't been moved yet. We're
		// ignoring errors here because there's nothing we can do about
		// them anyway.
		if !moved {
			os.Remove(name)
		}
	}(f, tmpName)

	// We'll try to apply the requested permissions. This may
	// not be effective on all platforms, but should at least work on
	// Unix-like targets and should be harmless elsewhere.
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("cannot set mode for temporary file %s: %s", tmpName, err)
	}

	// Write the credentials to the temporary file, then immediately close
	// it, whether or not the write succeeds. Note that closing the file here
	// is required because on Windows we can't move a file while it's open.
	_, err = f.Write(data)
	f.Close()
	if err != nil {
		return fmt.Errorf("cannot write to temporary file %s: %s", tmpName, err)
	}

	// Temporary file now replaces the original file, as atomically as
	// possible. (At the very least, we should not end up with a file
	// containing only a partial JSON object.)
	err = AtomicRename(tmpName, filename)
	if err != nil {
		return fmt.Errorf("failed to replace %s with temporary file %s: %s", filename, tmpName, err)
	}

	moved = true
	return nil
}
