// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateMigrateCommand is a Command implementation that migrates
// the state file from one location to another
type StateMigrateCommand struct {
	Meta

	// incompleteProviders is necessary here to coordinate separate
	// provider installation and lock file update processes.
	incompleteProviders []string
}

func (c *StateMigrateCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.Meta.View.Configure(common)

	args, diags := arguments.ParseStateMigrate(rawArgs)

	view := views.NewStateMigrate(args.ViewType, c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// FIXME: the -input flag value is needed but there is no clear path to pass
	// this value down, so we continue to mutate the Meta object state for now.
	c.Meta.input = args.InputEnabled

	// Command can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	if args.SourceLockFilePath != "" {
		if _, err := os.Stat(args.SourceLockFilePath); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unreadable source provider lock file",
				fmt.Sprintf("%q: %s", args.SourceLockFilePath, err.Error()),
			))
		}
	}

	// It is valid for the destination lockfile to be missing
	// while state exists - e.g. through the use of builtin provider
	// or outputs and use of a builtin backend
	// (as opposed to pluggable state store).
	if args.DestinationLockFilePath != "" {
		if _, err := os.Stat(args.DestinationLockFilePath); err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unreadable destination provider lock file",
				fmt.Sprintf("%q: %s", args.DestinationLockFilePath, err.Error()),
			))
		}
	}

	// return validation errors early if there are any
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	c.Meta.includeStateMigrateFiles = true
	dir := c.Meta.WorkingDir.RootModuleDir()
	cfg, mDiags := c.Meta.loadConfig(dir)
	if mDiags.HasErrors() {
		diags = diags.Append(mDiags)
		view.Diagnostics(diags)
		return 1
	}

	smi := cfg.Module.StateMigrationInstructions
	if smi == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No state migration instructions found",
			"No instructions were found in the configuration files. Please ensure that a file with a .tfmigrate.hcl extension is present and contains valid state migration instructions.",
		))
		view.Diagnostics(diags)
		return 1
	}

	var source string
	if smi.Backend != nil {
		source = fmt.Sprintf("backend %q", smi.Backend.Type)
	} else if smi.StateStore != nil {
		source = fmt.Sprintf("state store %q (%s)", smi.StateStore.Type,
			smi.StateStore.ProviderAddr)
	}

	rootMod := cfg.Module
	var destination string
	if rootMod.Backend != nil {
		destination = fmt.Sprintf("backend %q", rootMod.Backend.Type)
	} else if rootMod.StateStore != nil {
		destination = fmt.Sprintf("state store %q (%s)", rootMod.StateStore.Type,
			rootMod.StateStore.ProviderAddr)
	} else {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unknown migration destination",
			"No configuration was provided for where to migrate the state to. Please ensure that a file with a .tf extension is present and contains valid state_store or backend configuration inside the terraform block.",
		))
		view.Diagnostics(diags)
		return 1
	}

	view.Log("Migrating state from %s to %s...", source, destination)

	if smi.StateStore != nil || rootMod.StateStore != nil {
		// Dev overrides and unmanaged providers will influence the behaviour of state migrate
		// and the command's impact on the dependency lock file's contents, so warn users.
		diags = diags.Append(c.providerDevOverrideInitWarnings())
		diags = diags.Append(c.providerUnmanagedInitWarnings())
	}

	originalLocks, locksDiags := c.lockedDependencies()
	diags = diags.Append(locksDiags)
	if locksDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Load the source backend
	switch {
	case smi.Backend != nil:
		// TODO: Initialize the source backend
	case smi.StateStore != nil:
		// Initialize the source state_store

		// Get source provider requirements
		srcReq := make(providerreqs.Requirements, 1)
		srcReq[smi.StateStoreProvider.Type] = smi.StateStoreProvider.VersionConstraints

		// Conditionally use CLI flag to supplement the existing dependency locks.
		var extraLocks *depsfile.Locks
		if args.SourceLockFilePath != "" {
			// TODO - use file to set extraLocks, and also validate that the file contains the right provider version.
		}
		// Supplemented locks may include a new provider that isn't in the working directory's dependency lock file.
		// As that new lock describes the source state store provider, it will not be added to the dependency lock file
		// after a successful state migration.
		supplementedLocks := c.mergeLockedDependencies(originalLocks, extraLocks)

		upgrade := false // TODO - controlled by flag
		_, _, srcProviderDiags := c.getSingleProvider(ctx, srcReq, supplementedLocks, upgrade, MigrationSource, view)
		diags = diags.Append(srcProviderDiags)
		if srcProviderDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// TODO: Implement interactive prompt to use provider if it was just downloaded.
		// TODO: Implement equivalent for TF in automation.

		// TODO: Load the source state store
		// Use the lock returned from getSingleProvider to get a provider factory, then initialize the source state store with that factory.

		view.Log("Got %s locks ok.", MigrationSource)
	}

	// Load the destination backend
	var destinationLocks *depsfile.Locks // This may match the current dependency locks on disk, or may have a new lock for the destination provider added.
	switch {
	case rootMod.Backend != nil:
		// Initialize the destination backend
	case rootMod.StateStore != nil:
		// Initialize the destination state_store

		// Get required_providers entry in the configuration
		dstReq, dstReqDiags := c.getDestinationStateStoreProviderRequirements(rootMod.StateStore.ProviderAddr, rootMod.ProviderRequirements)
		diags = diags.Append(dstReqDiags)
		if dstReqDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// Conditionally use CLI flag to supplement the existing dependency locks.
		var extraLocks *depsfile.Locks
		if args.DestinationLockFilePath != "" {
			// TODO - use file to set extraLocks, and also validate that the file contains the right provider version.
		}
		// Supplemented locks may include a new provider that isn't in the working directory's dependency lock file.
		// As that new lock describes the destination state store provider, it will be added to the dependency lock file
		// after a successful state migration; the provider will be needed in subsequent commands.
		supplementedLocks := c.mergeLockedDependencies(originalLocks, extraLocks)

		upgrade := false // TODO - controlled by flag
		var dstProviderDiags tfdiags.Diagnostics
		// Returned value assigned to destinationLocks is used to update the dependency lock file after a successful state migration.
		_, destinationLocks, dstProviderDiags = c.getSingleProvider(ctx, dstReq, supplementedLocks, upgrade, MigrationDestination, view)
		diags = diags.Append(dstProviderDiags)
		if dstProviderDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// TODO: Implement interactive prompt to use provider if it was just downloaded.
		// TODO: Implement equivalent for TF in automation.

		// TODO: Load the destination state store
		// Use the lock to get a provider factory, then initialize the destination state store with that factory.

		view.Log("Got %s locks ok.", MigrationDestination)
	}

	// TODO: Perform the migration from source to destination

	// After a successful migration, the dependency lock file will be updated to include the destination backend's provider,
	// if it's new
	_, depLockFileDiags := c.saveDependencyLockFile(originalLocks, destinationLocks, view)
	diags = diags.Append(depLockFileDiags)
	if depLockFileDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(errors.New("Not implemented yet"))

	view.Diagnostics(diags)
	return 1
}

func (c *StateMigrateCommand) Help() string {
	helpText := `
Usage: terraform [global options] state migrate [options]

  Migrate state from source declared in the migration configuration (*.tfmigrate.hcl)
  to the destination declared in the root module (*.tf).

  An error will be returned if the migration fails, e.g. if the state
  is inaccessible or the migration configuration is invalid.

Options:

  -source-provider-lock-file       Path to a provider lock file for the source provider (requires -input=false).
                                   Defaults to using the working directory's .terraform.lock.hcl file.

  -destination-provider-lock-file  Path to a provider lock file for the destination provider (requires -input=false).
                                   Defaults to using the working directory's .terraform.lock.hcl file.

  -upgrade                         Trigger upgrade of the provider used for state storage.

  -input=true                      Enable input for interactive prompts (defaults to true, set to false in automation).
`
	return strings.TrimSpace(helpText)
}

func (c *StateMigrateCommand) Synopsis() string {
	return "Migrate the state from one location to another"
}

const (
	MigrationSource      = "source"
	MigrationDestination = "destination"
)

func (c *StateMigrateCommand) getDestinationStateStoreProviderRequirements(provider addrs.Provider, configReqs *configs.RequiredProviders) (providerreqs.Requirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	req := make(providerreqs.Requirements, 1)

	if configReqs == nil {
		panic(fmt.Sprintf("expected one provider requirement for the destination state store provider %q, but received empty data about required providers.", provider))
	}

	for providerReq := range maps.Values(configReqs.RequiredProviders) {
		if providerReq.Type.Equals(provider) {
			con, err := providerreqs.ParseVersionConstraints(providerReq.Requirement.Required.String())
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint syntax for state store provider",
					// The errors returned by ParseVersionConstraint already include
					// the section of input that was incorrect, so we don't need to
					// include that here.
					Detail:  fmt.Sprintf("Incorrect version constraint syntax: %s.", err.Error()),
					Subject: providerReq.Requirement.DeclRange.Ptr(),
				})
			}
			req[providerReq.Type] = con
		}
	}
	if len(req) != 1 {
		panic(fmt.Sprintf("expected exactly one provider requirement for the destination state store provider %q, got %d", provider, len(req)))
	}

	return req, diags
}

// getSingleProvider is used to download the source and/or destination state store providers during a state migration.
// Download of the up to 2 providers is kept separate due to:
// - Potential for downloading different versions of the same provider
// - Need to keep the locks separate for source and destination providers; destination providers are added to the dependency lock file.
func (c *StateMigrateCommand) getSingleProvider(ctx context.Context, reqs providerreqs.Requirements, locks *depsfile.Locks, upgrade bool, location string, view views.StateMigrate) (output bool, resultingLock *depsfile.Locks, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "install state migration "+location+" provider")
	defer span.End()

	// We expect to download only one provider
	if len(reqs) != 1 {
		panic(fmt.Sprintf("expected exactly one provider requirement for the destination state store provider, got %d", len(reqs)))
	}

	// Check for legacy provider addresses.
	for providerAddr := range reqs {
		if providerAddr.IsLegacy() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid legacy provider address",
				fmt.Sprintf(
					"This configuration or its associated state refers to the unqualified provider %q.\n\nYou must complete the Terraform 0.13 upgrade process before upgrading to later versions.",
					providerAddr.Type,
				),
			))
		}
	}
	if diags.HasErrors() {
		return false, nil, diags
	}

	// Use a source that looks for providers in all of the standard locations,
	// possibly customized by the user in CLI config.
	inst := c.providerInstaller()

	// Because we're currently just streaming a series of events sequentially
	// into the terminal, we're showing only a subset of the events to keep
	// things relatively concise. Later it'd be nice to have a progress UI
	// where statuses update in-place, but we can't do that as long as we
	// are shimming our vt100 output to the legacy console API on Windows.
	evts := &providercache.InstallerEvents{
		PendingProviders: func(reqs map[addrs.Provider]getproviders.VersionConstraints) {
			view.LogProviderInstallationMessage(views.InitializingStateMigrationProviderPluginMessage, location)
		},
		ProviderAlreadyInstalled: providerAlreadyInstalledCallback(view),
		BuiltInProviderAvailable: builtInProviderAvailableCallback(view),
		BuiltInProviderFailure:   builtInProviderFailureCallback(&diags),
		QueryPackagesBegin: func(provider addrs.Provider, versionConstraints getproviders.VersionConstraints, locked bool) {
			if locked {
				view.LogProviderInstallationMessage(views.ReusingPreviousVersionInfo, provider.ForDisplay())
			} else {
				if len(versionConstraints) > 0 {
					view.LogProviderInstallationMessage(views.FindingMatchingVersionMessage, provider.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints))
				} else {
					view.LogProviderInstallationMessage(views.FindingLatestVersionMessage, provider.ForDisplay())
				}
			}
		},
		LinkFromCacheBegin:   linkFromCacheBeginCallback(view),
		FetchPackageBegin:    fetchPackageBeginCallback(view),
		QueryPackagesFailure: queryPackagesFailureCallback(&diags, ctx, inst.ProviderSource(), reqs, nil),
		QueryPackagesWarning: queryPackagesWarningCallback(&diags),
		LinkFromCacheFailure: linkFromCacheFailureCallback(&diags),
		FetchPackageFailure:  fetchPackageFailureCallback(&diags, reqs),
		FetchPackageSuccess:  fetchPackageSuccessCallback(view),
		ProvidersLockUpdated: providersLockUpdatedCallback(&c.incompleteProviders),
		ProvidersFetched:     providersFetchedCallback(view),
	}
	ctx = evts.OnContext(ctx)

	mode := providercache.InstallNewProvidersOnly
	if upgrade {
		mode = providercache.InstallUpgrades
	}

	newLocks, err := inst.EnsureProviderVersions(ctx, locks, reqs, mode)
	if ctx.Err() == context.Canceled {
		diags = diags.Append(fmt.Errorf("Provider installation was canceled by an interrupt signal."))
		view.Diagnostics(diags)
		return true, nil, diags
	}
	if err != nil {
		// The errors captured in "err" should be redundant with what we
		// received via the InstallerEvents callbacks above, so we'll
		// just return those as long as we have some.
		if !diags.HasErrors() {
			diags = diags.Append(err)
		}

		return true, nil, diags
	}

	return true, newLocks, diags
}

// saveDependencyLockFile overwrites the contents of the dependency lock file.
// The calling code is expected to provide:
// 1. the previous locks (if any)
// 2. the lock for the destination state store provider (if any)
func (c *StateMigrateCommand) saveDependencyLockFile(previousLocks, dstProviderLocks *depsfile.Locks, view views.ProviderInstaller) (output bool, diags tfdiags.Diagnostics) {
	// Get the combination of locks from both potential provider download steps.
	newLocks := c.mergeLockedDependencies(previousLocks, dstProviderLocks)

	// If the provider dependencies have changed since the last run then we'll
	// say a little about that in case the reader wasn't expecting a change.
	if !newLocks.Equal(previousLocks) {
		// Jump in here and add a warning if any of the providers are incomplete.
		if len(c.incompleteProviders) > 0 {
			// We don't really care about the order here, we just want the
			// output to be deterministic.
			sort.Slice(c.incompleteProviders, func(i, j int) bool {
				return c.incompleteProviders[i] < c.incompleteProviders[j]
			})
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				incompleteLockFileInformationHeader,
				fmt.Sprintf(
					incompleteLockFileInformationBody,
					strings.Join(c.incompleteProviders, "\n  - "),
					getproviders.CurrentPlatform.String())))
		}
		if previousLocks.Empty() {
			// A change from empty to non-empty is special because it suggests
			// we're running "terraform init" for the first time against a
			// new configuration. In that case we'll take the opportunity to
			// say a little about what the dependency lock file is, for new
			// users or those who are upgrading from a previous Terraform
			// version that didn't have dependency lock files.
			view.LogProviderInstallationMessage(views.LockInfo)
			output = true
		} else {
			view.LogProviderInstallationMessage(views.DependenciesLockChangesInfo)
			output = true
		}
		lockFileDiags := c.replaceLockedDependencies(newLocks)
		diags = diags.Append(lockFileDiags)
	}
	return output, diags
}
