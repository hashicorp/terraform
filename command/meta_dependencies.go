package command

import (
	"os"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/tfdiags"
)

// dependenclyLockFilename is the filename of the dependency lock file.
//
// This file should live in the same directory as the .tf files for the
// root module of the configuration, alongside the .terraform directory
// as long as that directory's path isn't overridden by the TF_DATA_DIR
// environment variable.
//
// We always expect to find this file in the current working directory
// because that should also be the root module directory.
//
// Some commands have legacy command line arguments that make the root module
// directory something other than the root module directory; when using those,
// the lock file will be written in the "wrong" place (the current working
// directory instead of the root module directory) but we do that intentionally
// to match where the ".terraform" directory would also be written in that
// case. Eventually we will phase out those legacy arguments in favor of the
// global -chdir=... option, which _does_ preserve the intended invariant
// that the root module directory is always the current working directory.
const dependencyLockFilename = ".terraform.lock.hcl"

// lockedDependencies reads the dependency lock information from the lock file
// in the current working directory.
//
// If the lock file doesn't exist at the time of the call, lockedDependencies
// indicates success and returns an empty Locks object. If the file does
// exist then the result is either a representation of the contents of that
// file at the instant of the call or error diagnostics explaining some way
// in which the lock file is invalid.
//
// The result is a snapshot of the locked dependencies at the time of the call
// and does not update as a result of calling replaceLockedDependencies
// or any other modification method.
func (m *Meta) lockedDependencies() (*depsfile.Locks, tfdiags.Diagnostics) {
	// We check that the file exists first, because the underlying HCL
	// parser doesn't distinguish that error from other error types
	// in a machine-readable way but we want to treat that as a success
	// with no locks. There is in theory a race condition here in that
	// the file could be created or removed in the meantime, but we're not
	// promising to support two concurrent dependency installation processes.
	_, err := os.Stat(dependencyLockFilename)
	if os.IsNotExist(err) {
		return depsfile.NewLocks(), nil
	}

	return depsfile.LoadLocksFromFile(dependencyLockFilename)
}

// replaceLockedDependencies creates or overwrites the lock file in the
// current working directory to contain the information recorded in the given
// locks object.
func (m *Meta) replaceLockedDependencies(new *depsfile.Locks) tfdiags.Diagnostics {
	return depsfile.SaveLocksToFile(new, dependencyLockFilename)
}
