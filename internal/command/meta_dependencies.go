// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// dependencyLockFilename is the filename of the dependency lock file.
//
// This file should live in the same directory as the .tf files for the
// root module of the configuration, alongside the .terraform directory
// as long as that directory's path isn't overridden by the TF_DATA_DIR
// environment variable.
//
// We always expect to find this file in the current working directory
// because that should also be the root module directory.
const dependencyLockFilename = depsfile.LockFilePath // .terraform.lock.hcl

// lockedDependencies reads the dependency lock information from the default lock file location
// in the current working directory.
// Wraps the readLockedDependenciesFromPath method; see that method for details.
func (m *Meta) lockedDependencies() (*depsfile.Locks, tfdiags.Diagnostics) {
	return m.readLockedDependenciesFromPath(dependencyLockFilename)
}

// readLockedDependenciesFromPath reads the dependency lock information from the lock file at the given path.
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
func (m *Meta) readLockedDependenciesFromPath(filename string) (*depsfile.Locks, tfdiags.Diagnostics) {
	// We check that the file exists first, because the underlying HCL
	// parser doesn't distinguish that error from other error types
	// in a machine-readable way but we want to treat that as a success
	// with no locks. There is in theory a race condition here in that
	// the file could be created or removed in the meantime, but we're not
	// promising to support two concurrent dependency installation processes.
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return m.annotateDependencyLocksWithOverrides(depsfile.NewLocks()), nil
	}

	ret, diags := depsfile.LoadLocksFromFile(filename)
	return m.annotateDependencyLocksWithOverrides(ret), diags
}

// replaceLockedDependencies creates or overwrites the lock file in the
// current working directory to contain the information recorded in the given
// locks object.
//
// See saveDependencyLockFile, an opinionated wrapper for this method.
func (m *Meta) replaceLockedDependencies(new *depsfile.Locks) tfdiags.Diagnostics {
	return depsfile.SaveLocksToFile(new, dependencyLockFilename)
}

// mergeLockedDependencies combines two sets of locks. The 'base' locks are copied, and any providers
// present in the additional locks that aren't present in the base are added to that copy. The merged
// combination is returned.
//
// If you're combining locks derived from config with other locks (from state or deps locks file), then
// the config locks need to be the first argument to ensure that the merged locks contain any
// version constraints. Version constraint data is only present in configuration.
// This allows code in the init command to download providers in separate phases and
// keep the lock file updated accurately after each phase.
//
// This method supports downloading providers in 2 steps, and is used during the second download step and
// while updating the dependency lock file.
func (m *Meta) mergeLockedDependencies(baseLocks, additionalLocks *depsfile.Locks) *depsfile.Locks {
	var mergedLocks *depsfile.Locks
	if baseLocks != nil {
		mergedLocks = baseLocks.DeepCopy()
	} else {
		mergedLocks = depsfile.NewLocks()
	}

	// Append locks derived from the state to locks derived from config.
	for _, lock := range additionalLocks.AllProviders() {
		match := mergedLocks.Provider(lock.Provider())
		if match != nil {
			log.Printf("[TRACE] Ignoring provider %s version %s in mergeLockedDependencies; lock file contains %s provider already, at version %s",
				lock.Provider(),
				lock.Version(),
				match.Provider(),
				match.Version(),
			)
		} else {
			// This is a new provider now present in the lockfile yet
			log.Printf("[DEBUG] Appending provider %s to the lock file", lock.Provider())
			mergedLocks.SetProvider(lock.Provider(), lock.Version(), lock.VersionConstraints(), lock.AllHashes())
		}
	}

	// Override the locks file with the new combination of locks
	return mergedLocks
}

// annotateDependencyLocksWithOverrides modifies the given Locks object in-place
// to track as overridden any provider address that's subject to testing
// overrides, development overrides, or "unmanaged provider" status.
//
// This is just an implementation detail of the lockedDependencies method,
// not intended for use anywhere else.
func (m *Meta) annotateDependencyLocksWithOverrides(ret *depsfile.Locks) *depsfile.Locks {
	if ret == nil {
		return ret
	}

	for addr := range m.ProviderDevOverrides {
		log.Printf("[DEBUG] Provider %s is overridden by dev_overrides", addr)
		ret.SetProviderOverridden(addr)
	}
	for addr := range m.UnmanagedProviders {
		log.Printf("[DEBUG] Provider %s is overridden as an \"unmanaged provider\"", addr)
		ret.SetProviderOverridden(addr)
	}
	if m.testingOverrides != nil {
		for addr := range m.testingOverrides.Providers {
			log.Printf("[DEBUG] Provider %s is overridden in Meta.testingOverrides", addr)
			ret.SetProviderOverridden(addr)
		}
	}

	return ret
}

// saveDependencyLockFile can overwrite the contents of the dependency lock file.
// If the locks match the previous locks, then the file is not updated and no output is produced.
// If a "readonly" -lockfile flag is supplied then changing the file is blocked.
func (m *Meta) saveDependencyLockFile(previousLocks, newLocks *depsfile.Locks, incompleteProviders []string, flagLockfile string, view views.ProviderInstaller) (output bool, diags tfdiags.Diagnostics) {
	// If the provider dependencies have changed since the last run then we'll
	// say a little about that in case the reader wasn't expecting a change.
	// (When we later integrate module dependencies into the lock file we'll
	// probably want to refactor this so that we produce one lock-file related
	// message for all changes together, but this is here for now just because
	// it's the smallest change relative to what came before it, which was
	// a hidden JSON file specifically for tracking providers.)
	if !newLocks.Equal(previousLocks) {
		// if readonly mode
		if flagLockfile == "readonly" {
			// check if required provider dependencies change
			if !newLocks.EqualProviderAddress(previousLocks) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					`Provider dependency changes detected`,
					`Changes to the required provider dependencies were detected, but the lock file is read-only. To use and record these requirements, run "terraform init" without the "-lockfile=readonly" flag.`,
				))
				return output, diags
			}
			// suppress updating the file to record any new information it learned,
			// such as a hash using a new scheme.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				`Provider lock file not updated`,
				`Changes to the provider selections were detected, but not saved in the .terraform.lock.hcl file. To record these selections, run "terraform init" without the "-lockfile=readonly" flag.`,
			))
			return output, diags
		}
		// Jump in here and add a warning if any of the providers are incomplete.
		if len(incompleteProviders) > 0 {
			// We don't really care about the order here, we just want the
			// output to be deterministic.
			sort.Slice(incompleteProviders, func(i, j int) bool {
				return incompleteProviders[i] < incompleteProviders[j]
			})
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				incompleteLockFileInformationHeader,
				fmt.Sprintf(
					incompleteLockFileInformationBody,
					strings.Join(incompleteProviders, "\n  - "),
					getproviders.CurrentPlatform.String())))
		}
		if previousLocks.Empty() {
			// A change from empty to non-empty is special because it suggests
			// we're running "terraform init" for the first time against a
			// new configuration. In that case we'll take the opportunity to
			// say a little about what the dependency lock file is, for new
			// users or those who are upgrading from a previous Terraform
			// version that didn't have dependency lock files.
			view.LockfileCreated()
			output = true
		} else {
			view.LockfileUpdated()
			output = true
		}
		lockFileDiags := m.replaceLockedDependencies(newLocks)
		diags = diags.Append(lockFileDiags)
	}
	return output, diags
}
