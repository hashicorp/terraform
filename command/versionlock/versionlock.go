// Package versionlock deals with the .terraform-version lock file convention
// for recording which Terraform CLI version a particular directory is currently
// intending to work with.
//
// The goal here is that switching to a new Terraform version is a decision
// made together by a whole team, via a process like pull requests. Terraform
// CLI itself enforces .terraform-version lock files containing exact version
// numbers, refusing to operate in a directory that selects a conflicting
// version.
//
// .terraform-version is a separate text file containing only a version number.
// This convention is intended to be simple enough that it can also be consumed
// by separate version manager wrapper scripts, which are often implemented
// in shell scripting languages.
//
// Terraform will look for a file named .terraform-version in the current
// working directory and any ancestor directories in the current
// volume/filesystem. A file containing a string that isn't a valid version
// number is interpreted as no restriction at all, without continuing to
// ancestor directories, to allow for existing version manager scripts that
// interpret .terraform-version in a more liberal way.
package versionlock

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	version "github.com/hashicorp/go-version"
)

// LockFileName is the filename that this package will search for or generate.
const LockFileName = ".terraform-version"

// GetLockedCLIVersion returns the exact CLI version locked for a particular
// directory, or nil if that directory does not have an exact version lock.
//
// If the version number is non-nil, GetLockedCLIVersion also returns the
// path to the file that made the final determination, so that the caller might
// mention it in an error message it produces.
//
// This function always succeeds because if it encounters any filesystem errors
// during its work it will interpret that as having no locked version.
func GetLockedCLIVersion(baseDir string) (locked *version.Version, decidingFile string) {
	// We'll use an absolute version of the given directory so that
	// we can use syntax-only path manipulation to walk up the filesystem
	// heirarchy all the way to the root.
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, ""
	}

	// Start in the given directory. We'll walk up from here.
	dir := absDir
	for {
		possibleFilename := filepath.Join(dir, LockFileName)

		raw, err := ioutil.ReadFile(possibleFilename)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, possibleFilename
			}

			parentDir, _ := filepath.Split(dir)
			parentDir = filepath.Clean(parentDir) // trim trailing slash on non-root, and any other normalization

			// If we're not making progress then we've hit a root directory and
			// so we'll give up.
			if parentDir == dir {
				return nil, possibleFilename
			}

			dir = parentDir
			continue // ignore "file not found" and continue walking up
		}

		// If we get here then we've found a .terraform-version file that
		// we were able to open and read, so we'll either return the exact
		// version number it indicates or return nil (indicating "no locked
		// version") if we can't parse it.
		//
		// If we get a parse error we assume it's one of the inexact patterns
		// supported by tfenv, in which case the user is presumably intending
		// to have tfenv choose a version automatically and so it would be
		// annoying if we then tried to enforce a different exact version
		// number chosen by an ancestor directory.
		locked, err = version.NewVersion(strings.TrimSpace(string(raw)))
		if err != nil {
			// Invalid syntax is understood as no selection at all.
			return nil, possibleFilename
		}
		return locked, possibleFilename
	}
}

// SetLockedCLIVersion creates or updates a version lock file for the given
// base directory.
func SetLockedCLIVersion(baseDir string, locked *version.Version) error {
	return errors.New("not yet implemented")
}
