// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package replacefile

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWriteFile uses a temporary file along with this package's AtomicRename
// function in order to provide a replacement for os.WriteFile that
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
	f, err := os.CreateTemp(dir, file) // alongside target file and with a similar name
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

// NonAtomicWriteFileWithBackup creates a backup file containing the original content of the target file and then updates
// the target file with new content, in order to provides a replacement for os.WriteFile that ensures that there is no data
// loss if the process is disrupted.
//
// This logic is based on implementation of the fmt command in the Go standard library.
// See: https://cs.opensource.google/go/go/+/master:src/cmd/gofmt/gofmt.go;l=468;drc=d98516a9d2f88cc0d6e88849e1a8f4e4f1e6f465
//
// Whereas AtomicWriteFile promises that the file at the given filename will either contain the entirety of the previous contents
// or the entirety of the given data array if opened and read at any point during its execution, NonAtomicWriteFileWithBackup
// is not implemented to be atomic in that sense. However, data loss is still prevented.
//
// The possible outcomes are:
// 1. The file is successfully updated with the new content.
// 2. The update fails, but the original content is unchanged in or successfully restored to the target file.
// 3. Both the update and the restoration fails somehow, but the backup file still exists for manual recovery
//
// The backup file will be cleaned up when the function can guarantee that original or updated data can be found in the target file,
// but if an error prevents that the backup will be left for users to manually recover the original content.
//
// NonAtomicWriteFileWithBackup updates the original file with new data, whereas AtomicWriteFile replaces the original file.
// This means that metadata on the original file can be preserved when using NonAtomicWriteFileWithBackup,
// e.g. ownership, file creation timestamp.
func NonAtomicWriteFileWithBackup(filename string, originalData, formattedData []byte, perm os.FileMode) error {
	// writeFailError produces an error stating the overall file change could not be fulfilled.
	// The returned error will either wrap 1 or 2 errors:
	// 1. The error that occurred while writing to the target file.
	// 2. The error that occurred while attempting to restore the original content from the backup file, if applicable.
	writeFailError := func(writeError, restoreError error) error {
		if restoreError != nil {
			return fmt.Errorf("Failed to write %s: %w; additionally, error restoring original content: %v", filename, writeError, restoreError)
		}
		return fmt.Errorf("Failed to write %s: %w", filename, writeError)
	}

	dir := filepath.Dir(filename)

	// Create a backup temporary file that contains the original content of the file.
	backup, err := os.CreateTemp(dir, filepath.Base(filename))
	if err != nil {
		errExtra := fmt.Errorf("error creating and opening temporary backup file for %s: %w", filename, err)
		return writeFailError(errExtra, nil)
	}
	defer backup.Close()

	_, err = backup.Write(originalData)
	if err != nil {
		os.Remove(backup.Name())
		errExtra := fmt.Errorf("error writing to backup temporary file %s: %w", backup.Name(), err)
		return writeFailError(errExtra, nil)
	}

	// Open the target file, attempt to write the formatted data to it
	f, err := os.OpenFile(filename, os.O_WRONLY, perm)
	if err != nil {
		os.Remove(backup.Name())
		errExtra := fmt.Errorf("error opening target file %s: %w", filename, err)
		return writeFailError(errExtra, nil)
	}

	n, err := f.Write(formattedData)
	if err == nil {
		err = f.Truncate(int64(n))
	}

	// restoreFailError produces an error describing failure to restore original content to the target file.
	// restoreFailError is intended to be used to create the second argument for writeFailError.
	restoreFailError := func(e error) error {
		return fmt.Errorf("error restoring file %s to original: %v; original content backed up in %s\n", filename, e, backup.Name())
	}

	if err != nil {
		// In response to an error we'll attempt to restore the original content
		// to the target file. If that fails, the original content is still
		// available in the backup temporary file.

		if n == 0 {
			// The original file was unchanged; backup not needed
			os.Remove(backup.Name())
			return writeFailError(fmt.Errorf("file %s unchanged; error while writing to file %s: %s", filename, filename, err), nil)
		}

		// Try to restore the original content
		no, erro := f.WriteAt(originalData, 0)
		if erro != nil {
			return writeFailError(err, restoreFailError(erro))
		}

		if no < n {
			// The original file is shorter; truncate
			if erro := f.Truncate(int64(no)); erro != nil {
				return writeFailError(err, restoreFailError(erro))
			}
		}

		if erro := f.Close(); erro != nil {
			return writeFailError(err, restoreFailError(erro))
		}

		// We successfully restored the original content to the file,
		// but still need to report the original write failure.
		os.Remove(backup.Name())
		return writeFailError(err, nil)
	}

	if err := f.Close(); err != nil {
		return writeFailError(err, nil)
	}

	// The file was successfully updated; remove backup
	os.Remove(backup.Name())
	return nil
}
