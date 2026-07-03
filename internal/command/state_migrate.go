// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/workdir"
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
	if cfg.Module.StateStore != nil {
		cfg.Module.StateStore.ProviderSupplyMode = c.getProviderSupplyModeForStateStore(cfg.Module)
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

	// When configuring the source and destination backends,
	// we prepare options for the migration action.
	migrateOpts := &backendMigrateOpts{
		ViewType: args.ViewType,
	}

	// Load the source backend
	var source string
	var sourceLock *depsfile.Locks // This should only contain a single lock, if non nil. Used to avoid re-download if destination provider is the same.
	if smi.Backend != nil {
		source = fmt.Sprintf("backend %q", smi.Backend.Type)

		srcB, _, srcDiags := c.Meta.backendInitFromConfig(smi.Backend)
		diags = diags.Append(srcDiags)
		if !diags.HasErrors() {
			migrateOpts.SourceType = smi.Backend.Type
			migrateOpts.Source = srcB
		}
	} else if smi.StateStore != nil {
		source = fmt.Sprintf("state store %q (%s)", smi.StateStore.Type,
			smi.StateStore.ProviderAddr.ForDisplay())

		// Load any pre-existing source provider lock file.
		srcLocks, srcLockDiags := c.readLockedDependenciesFromPath(args.SourceLockFilePath)
		diags = diags.Append(srcLockDiags)
		if srcLockDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		upgrade := false // The first provider download step will never be an upgrade. Either it's constrained by a preexisting lock or there is no lock.
		var srcProviderDiags tfdiags.Diagnostics
		var output bool
		output, sourceLock, srcProviderDiags = c.getSingleProvider(ctx, smi.StateStore.Type, smi.StateStoreProvider.Requirement, srcLocks, upgrade, MigrationSource, view)
		diags = diags.Append(srcProviderDiags)
		if srcProviderDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if output {
			// Space out provider download output from the migration output below.
			view.Spacer()
		}

		srcB, _, _, srcDiags := c.Meta.stateStoreInitFromConfig(smi.StateStore, sourceLock)
		diags = diags.Append(srcDiags)
		if !diags.HasErrors() {
			migrateOpts.SourceType = smi.StateStore.Type
			migrateOpts.Source = srcB
		}
	}

	// Load the destination backend
	rootMod := cfg.Module
	var destination string
	var destinationLock *depsfile.Locks                               // This should only contain a single lock, if non nil. Used to update the dependency lock file on disk.
	var bsf *workdir.BackendStateFile = workdir.NewBackendStateFile() // Collect data below for updating the backend state file.
	if rootMod.Backend != nil {
		destination = fmt.Sprintf("backend %q", rootMod.Backend.Type)

		dstB, dstConfig, dstDiags := c.Meta.backendInitFromConfig(rootMod.Backend)
		diags = diags.Append(dstDiags)
		if !diags.HasErrors() {
			// Assign migration options for the migration action later in the command
			migrateOpts.DestinationType = rootMod.Backend.Type
			migrateOpts.Destination = dstB

			// Capture details of the destination backend for updating the backend state file after a successful migration.
			_, cHash, bcDiags := c.backendConfig(&BackendOpts{
				BackendConfig: rootMod.Backend,
			})
			diags = diags.Append(bcDiags)
			if bcDiags.HasErrors() {
				view.Diagnostics(diags)
				return 1
			}
			bsf.Backend = &workdir.BackendConfigState{
				Type: rootMod.Backend.Type,
				Hash: uint64(cHash),
			}
			err := bsf.Backend.SetConfig(dstConfig, dstB.ConfigSchema())
			if err != nil {
				diags = diags.Append(fmt.Errorf("Can't serialize backend configuration as JSON: %s", err))
				view.Diagnostics(diags)
				return 1
			}
		}
	} else if rootMod.StateStore != nil {
		destination = fmt.Sprintf("state store %q (%s)", rootMod.StateStore.Type,
			rootMod.StateStore.ProviderAddr.ForDisplay())

		// Get single required_providers entry for state store provider.
		dstReq, dstReqDiags := c.getDestinationStateStoreProviderRequirements(rootMod.StateStore.ProviderAddr, rootMod.ProviderRequirements)
		diags = diags.Append(dstReqDiags)
		if dstReqDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// Load any pre-existing destination provider lock file.
		dstLocks, dstLockDiags := c.readLockedDependenciesFromPath(args.DestinationLockFilePath)
		diags = diags.Append(dstLockDiags)
		if dstLockDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// The source provider download step may have introduced a new lock that can be re-used here.
		// Else, this download step could re-download the same provider if the migration is between stores
		// in the same provider.
		//
		// TODO: Make this conditional based on whether we're doing an upgrade or not?
		//       Or is use of upgrade flag in second download sufficient?
		var mergedLocks *depsfile.Locks
		if sourceLock != nil {
			mergedLocks = c.mergeLockedDependencies(dstLocks, sourceLock)
		} else {
			mergedLocks = dstLocks
		}

		// Perform download of the destination provider.
		// This may be controlled by a pre-existing lock from above or not, therefore the returned
		// lock for the destination state store may not already be in the lock file.
		//
		// We only pass in a single required provider, so we expect a single lock to be
		// returned. This will be added the dependency lock file after a successful migration.
		upgrade := false // TODO - control this by -upgrade flag
		var dstProviderDiags tfdiags.Diagnostics
		var output bool
		output, destinationLock, dstProviderDiags = c.getSingleProvider(ctx, rootMod.StateStore.Type, dstReq, mergedLocks, upgrade, MigrationDestination, view)
		diags = diags.Append(dstProviderDiags)
		if dstProviderDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if output {
			// Space out provider download output from the migration output below.
			view.Spacer()
		}

		dstB, stateStoreConfigVal, providerConfigVal, dstDiags := c.Meta.stateStoreInitFromConfig(rootMod.StateStore, destinationLock)
		diags = diags.Append(dstDiags)
		if !diags.HasErrors() {
			migrateOpts.DestinationType = rootMod.StateStore.Type
			migrateOpts.Destination = dstB
		}

		// Capture details of the destination state store for updating the backend state file after a successful migration.
		_, cHash, sscDiags := c.stateStoreConfig(&BackendOpts{
			StateStoreConfig: rootMod.StateStore,
			Locks:            destinationLock,
		})
		diags = diags.Append(sscDiags)
		if sscDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		v := destinationLock.Provider(rootMod.StateStore.ProviderAddr).Version() // We just downloaded this provider, so the lock wil be present.
		version, err := providerreqs.GoVersionFromVersion(v)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to convert provider version to Go version: %s", err))
			view.Diagnostics(diags)
			return 1
		}

		bsf.StateStore = &workdir.StateStoreConfigState{
			Type: rootMod.StateStore.Type,
			Hash: uint64(cHash),
			Provider: &workdir.ProviderConfigState{
				Source:  &rootMod.StateStore.ProviderAddr,
				Version: version,
			},
			ProviderSupplyMode: rootMod.StateStore.ProviderSupplyMode,
		}
		err = bsf.StateStore.SetConfig(stateStoreConfigVal, dstB.ConfigSchema())
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to set state store configuration: %w", err))
			view.Diagnostics(diags)
			return 1
		}

		err = bsf.StateStore.Provider.SetConfig(providerConfigVal, dstB.ProviderSchema())
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to set state store provider configuration: %w", err))
			view.Diagnostics(diags)
			return 1
		}

	} else {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unknown migration destination",
			"No configuration was provided for where to migrate the state to. Please ensure that a file with a .tf extension is present and contains valid state_store or backend configuration inside the terraform block.",
		))
	}

	// present all errors from above together so user can fix them all at once
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	view.Log("[reset][bold]Migrating state from %s to %s...[reset]", source, destination)

	// Perform the migration from source to destination
	err := c.Meta.backendMigrateState(migrateOpts)
	if err != nil {
		diags = diags.Append(fmt.Errorf("migration failed: %w", err))
		view.Diagnostics(diags)
		return 1
	}

	// After a successful migration to a state store, we must make sure the dependency lock file contains the
	// details of the destination state store provider.
	if rootMod.StateStore != nil {
		originalLocks, originalLockDiags := c.lockedDependencies()
		diags = diags.Append(originalLockDiags)
		if originalLockDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// Get the combination of locks
		//
		// Take the lock from the destination provider download and add in the original locks from the dependency lock file.
		// This means the lock created from the destination provider download is authoritative (e.g. any hashes from the
		// installation process are preserved)
		newLocks := c.mergeLockedDependencies(destinationLock, originalLocks)

		output, depLockFileDiags := c.saveDependencyLockFile(originalLocks, newLocks, view)
		diags = diags.Append(depLockFileDiags)
		if depLockFileDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if output {
			// Space out lock file creation/update output from the final outputs below.
			view.Spacer()
		}
	}

	// Success, so update backend state file to reflect the new location state is stored in.
	bsfDiags := c.updateBackendStateFile(bsf)
	diags = diags.Append(bsfDiags)
	if bsfDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	view.Diagnostics(diags) // Log any warnings

	view.Log("[reset][bold]Finished migrating state from %s to %s.[reset]", source, destination)

	return 0
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

func (c *StateMigrateCommand) updateBackendStateFile(s *workdir.BackendStateFile) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	statePath := filepath.Join(c.DataDir(), DefaultStateFilename)
	sMgr := &clistate.LocalState{Path: statePath}
	if err := sMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load the backend state file when preparing to update it: %s", err))
		return diags
	}

	if err := sMgr.WriteState(s); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return diags
	}
	if err := sMgr.PersistState(); err != nil {
		diags = diags.Append(errBackendWriteSavedDiag(err))
		return diags
	}

	return diags
}

func (c *StateMigrateCommand) getDestinationStateStoreProviderRequirements(provider addrs.Provider, configReqs *configs.RequiredProviders) (providerreqs.Requirements, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	req := make(providerreqs.Requirements, 1)

	if configReqs == nil {
		panic(fmt.Sprintf("expected one provider requirement for the destination state store provider %q, but received empty data about required providers.", provider))
	}

	for _, providerReq := range configReqs.RequiredProviders {
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

// saveDependencyLockFile overwrites the contents of the dependency lock file.
func (c *StateMigrateCommand) saveDependencyLockFile(previousLocks, newLocks *depsfile.Locks, view views.StateMigrate) (output bool, diags tfdiags.Diagnostics) {
	// The state migrate command does not support the -lockfile=readonly flag
	// This flag is specific to the init command, and can only take "" or "readonly" as values.
	// As state migrate doesn't take this flag, we can safely set it to "" here.
	flagLockfile := ""

	return c.Meta.saveDependencyLockFile(previousLocks, newLocks, c.incompleteProviders, flagLockfile, view)
}

// getSingleProvider is used to download the source and/or destination state store providers during a state migration.
// Download of the up to 2 providers is kept separate due to:
// - Potential for downloading different versions of the same provider
// - Need to keep the locks separate for source and destination providers; destination providers are added to the dependency lock file.
func (c *StateMigrateCommand) getSingleProvider(ctx context.Context, storeName string, reqs providerreqs.Requirements, locks *depsfile.Locks, upgrade bool, location string, view views.StateMigrate) (output bool, resultingLock *depsfile.Locks, diags tfdiags.Diagnostics) {
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
			view.LogInitMessage(views.InitializingStateStoreProviderPluginMessage, storeName)
		},
		ProviderAlreadyInstalled: providerAlreadyInstalledCallback(view),
		BuiltInProviderAvailable: builtInProviderAvailableCallback(view),
		BuiltInProviderFailure:   builtInProviderFailureCallback(&diags),
		QueryPackagesBegin: func(provider addrs.Provider, versionConstraints getproviders.VersionConstraints, locked bool) {
			if locked {
				view.LogInitMessage(views.ReusingPreviousVersionInfo, provider.ForDisplay())
			} else {
				if len(versionConstraints) > 0 {
					view.LogInitMessage(views.FindingMatchingVersionMessage, provider.ForDisplay(), getproviders.VersionConstraintsString(versionConstraints))
				} else {
					view.LogInitMessage(views.FindingLatestVersionMessage, provider.ForDisplay())
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
