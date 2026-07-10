// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func (c *InitCommand) run(initArgs *arguments.Init, view views.Init) int {
	var diags tfdiags.Diagnostics

	c.forceInitCopy = initArgs.ForceInitCopy
	c.Meta.stateLock = initArgs.StateLock
	c.Meta.stateLockTimeout = initArgs.StateLockTimeout
	c.reconfigure = initArgs.Reconfigure
	c.migrateState = initArgs.MigrateState
	c.Meta.ignoreRemoteVersion = initArgs.IgnoreRemoteVersion
	c.Meta.input = initArgs.InputEnabled
	c.Meta.targetFlags = initArgs.TargetFlags
	c.Meta.compactWarnings = initArgs.CompactWarnings

	// Copying the state only happens during backend migration, so setting
	// -force-copy implies -migrate-state
	if c.forceInitCopy {
		c.migrateState = true
	}

	if len(initArgs.PluginPath) > 0 {
		c.pluginPath = initArgs.PluginPath
	}

	// Get the working directory
	path := c.Meta.WorkingDir.RootModuleDir()

	if err := c.storePluginPath(c.pluginPath); err != nil {
		diags = diags.Append(fmt.Errorf("Error saving -plugin-dir to workspace directory: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Initialization can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	if initArgs.FromModule != "" {
		src := initArgs.FromModule

		empty, err := configs.IsEmptyDir(path, initArgs.TestsDirectory)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error validating destination directory: %s", err))
			view.Diagnostics(diags)
			return 1
		}
		if !empty {
			diags = diags.Append(errors.New(strings.TrimSpace(errInitCopyNotEmpty)))
			view.Diagnostics(diags)
			return 1
		}

		view.Output(views.CopyingConfigurationMessage, src)

		hooks := uiModuleInstallHooks{
			Ui:             c.Ui,
			ShowLocalPaths: false, // since they are in a weird location for init
			View:           view,
		}

		ctx, span := tracer.Start(ctx, "-from-module=...", trace.WithAttributes(
			attribute.String("module_source", src),
		))

		initDirFromModuleAbort, initDirFromModuleDiags := c.initDirFromModule(ctx, path, src, hooks)
		diags = diags.Append(initDirFromModuleDiags)
		if initDirFromModuleAbort || initDirFromModuleDiags.HasErrors() {
			view.Diagnostics(diags)
			span.SetStatus(codes.Error, "module installation failed")
			span.End()
			return 1
		}
		span.End()

		view.Output(views.EmptyMessage)
	}

	// If our directory is empty, then we're done. We can't get or set up
	// the backend with an empty directory.
	empty, err := configs.IsEmptyDir(path, initArgs.TestsDirectory)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Error checking configuration: %s", err))
		view.Diagnostics(diags)
		return 1
	}
	if empty {
		view.Output(views.OutputInitEmptyMessage)
		return 0
	}

	// Load just the root module to begin backend and module initialization
	rootModEarly, earlyConfDiags := c.loadSingleModuleWithTests(path, initArgs.TestsDirectory)

	// There may be parsing errors in config loading but these will be shown later _after_
	// checking for core version requirement errors. Not meeting the version requirement should
	// be the first error displayed if that is an issue, but other operations are required
	// before being able to check core version requirements.
	if rootModEarly == nil {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)), earlyConfDiags)
		view.Diagnostics(diags)

		return 1
	}
	if !(c.Meta.AllowExperimentalFeatures && initArgs.EnablePssExperiment) && rootModEarly.StateStore != nil {
		// TODO(SarahFrench/radeksimko) - remove when this feature isn't experimental.
		// This approach for making the feature experimental is required
		// to let us assert the feature is gated behind an experiment in tests.
		// See https://github.com/hashicorp/terraform/pull/37350#issuecomment-3168555619

		detail := "Pluggable state store is an experiment which requires"
		if !c.Meta.AllowExperimentalFeatures {
			detail += " an experimental build of terraform"
		}
		if !initArgs.EnablePssExperiment {
			if !c.Meta.AllowExperimentalFeatures {
				detail += " and"
			}
			detail += " -enable-pluggable-state-storage-experiment flag"
		}

		diags = diags.Append(earlyConfDiags)
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Pluggable state store experiment not supported",
			Detail:   detail,
			Subject:  &rootModEarly.StateStore.TypeRange,
		})
		view.Diagnostics(diags)

		return 1
	}

	// Load locks from any pre-existing dependency lock file.
	previousLocks, locksDiags := c.lockedDependencies()
	diags = diags.Append(locksDiags)
	if locksDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	var pssLock *depsfile.Locks // May end up containing 0 or 1 lock, and needs to be able to influence `getProviders` below.
	if rootModEarly.StateStore != nil {
		// If the user supplies -state-provider-lock-file to init then we need to let those locks influence provider installation.
		// `alteredPreviousLocks` will only be different from the locks loaded from the working directory if the user supplied a supplementary lock file via -state-provider-lock-file.
		alteredPreviousLocks := previousLocks.DeepCopy()

		if initArgs.StateStoreProviderLockFile != "" {
			stateStoreLocks, lockDiags := c.readLockedDependenciesFromPath(initArgs.StateStoreProviderLockFile)
			if lockDiags.HasErrors() {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Error loading -state-provider-lock-file lock file",
					fmt.Sprintf("Terraform experienced an error loading the file at %q: %s", initArgs.StateStoreProviderLockFile, lockDiags.Err()),
				))
				view.Diagnostics(diags)
				return 1
			}
			diags = diags.Append(lockDiags) // capture any warnings

			lock := stateStoreLocks.Provider(rootModEarly.StateStore.ProviderAddr)
			if lock == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"State store provider not described in dependency lock file supplied via -state-provider-lock-file flag",
					fmt.Sprintf("Terraform checked the lock file at %q, supplied via the -state-provider-lock-file flag, but could not find the state store provider %q (%s). To get a sufficient lock file create a minimal configuration with the specific provider and version you want to use described in a required_providers block. Then, perform \"terraform init\" manually to create a dependency lock file describing that provider. After checking the lock file's contents you can retry the original command that produced this error by running: \"terraform init -input=false -state-provider-lock-file=<path to the newly-created lock file>\".",
						initArgs.StateStoreProviderLockFile,
						rootModEarly.StateStore.ProviderAddr.Type,
						rootModEarly.StateStore.ProviderAddr.ForDisplay(),
					),
				))
				view.Diagnostics(diags)
				return 1
			}

			// Overwrite or add the state store provider lock to the other locks for this project
			alteredPreviousLocks.SetProvider(
				lock.Provider(),
				lock.Version(),
				lock.VersionConstraints(),
				lock.PreferredHashes(),
			)
		}

		// The init command is not allowed to upgrade the provider used for state storage
		// We warn that upgrades will not impact the provider, and upgrades will only work via `terraform state migrate -upgrade`.
		var allowUpgrade bool
		if initArgs.Upgrade {
			if initArgs.Reconfigure {
				allowUpgrade = true // user is opting out of migrating state; whatever happens, happens
			} else {
				allowUpgrade = false // the installer will only be able to reuse the old version.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Cannot upgrade the provider used for state storage during \"terraform init -upgrade\"",
					fmt.Sprintf(`Terraform will not upgrade the %s (%q) provider as part of this operation because it is used for state storage.

Please use \"terraform state migrate -upgrade\" to upgrade the state store provider and navigate migrating your state between the two versions.`,
						rootModEarly.StateStore.ProviderAddr.Type,
						rootModEarly.StateStore.ProviderAddr.ForDisplay(),
					),
				),
				)
			}
		}

		var configProvidersOutput bool
		var safeInstallAction SafeStateStoreProviderInstallAction
		var stateStoreProviderAuthResult *getproviders.PackageAuthenticationResult
		var configProviderDiags tfdiags.Diagnostics
		configProvidersOutput, pssLock, safeInstallAction, stateStoreProviderAuthResult, configProviderDiags = c.getProvidersFromPSSConfig(ctx, rootModEarly, alteredPreviousLocks, allowUpgrade, initArgs.PluginPath, initArgs.Lockfile, view)
		diags = diags.Append(configProviderDiags)
		if configProviderDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if configProvidersOutput {
			// If we outputted information, then we need to output a newline
			// so that our success message is nicely spaced out from prior text.
			view.Output(views.EmptyMessage)
		}

		// Course of action depends on the SafeStateStoreProviderInstallAction returned from getProvidersFromPSSConfig
		safeDiags := c.handleSafeProviderInstallAction(safeInstallAction, rootModEarly.StateStore.ProviderAddr, stateStoreProviderAuthResult, pssLock, alteredPreviousLocks, initArgs.StateStoreProviderLockFile, c, view)
		diags = diags.Append(safeDiags)
		if safeDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}

		// Record how the state store provider is supplied to Terraform
		rootModEarly.StateStore.ProviderSupplyMode = c.Meta.getProviderSupplyModeForStateStore(rootModEarly)
		if rootModEarly.StateStore.ProviderSupplyMode == getproviders.Unset {
			panic("unset provider supply mode for state store")
		}
	}

	var back backend.Backend

	var backDiags tfdiags.Diagnostics
	var backendOutput bool
	switch {
	case initArgs.Cloud && rootModEarly.CloudConfig != nil:
		back, backendOutput, backDiags = c.initCloud(ctx, rootModEarly, initArgs.BackendConfig, initArgs.ViewType, view)
	case initArgs.Backend:
		back, backendOutput, backDiags = c.initBackend(ctx, rootModEarly, initArgs, pssLock, view)
	default:
		// load the previously-stored backend config
		back, backDiags = c.Meta.backendFromState(ctx)
	}
	if backendOutput {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// Set up the policy client now that the backend is configured, so the
	// entitlement can be read from it (as plan and apply do). getModules and
	// getProviders below consume the client through the provider hook.
	var policyClient policy.Client
	if len(initArgs.PolicyPaths) > 0 {
		var policyDiags policy.Diagnostics
		var stopClient func()
		policyClient, policyDiags, stopClient = c.PolicyClient(ctx, initArgs.PolicyPaths, backendPolicyEntitlement(back))
		defer stopClient()
		// Stream any policy setup diagnostics (e.g. a failure to connect to the
		// policy engine).
		view.PolicyDiagnostics(policyDiags)
		if policyDiags.HasErrors() {
			diags = diags.Append(earlyConfDiags)
			diags = diags.Append(backDiags)
			view.Diagnostics(diags)
			return 1
		}
	}
	providerHook := &providerPolicyHook{
		client:     policyClient,
		view:       view,
		rootModule: rootModEarly,
	}

	var state *states.State

	// If we have a functional backend (either just initialized or initialized
	// on a previous run) we'll use the current state as a potential source
	// of provider dependencies.
	if back != nil {
		c.ignoreRemoteVersionConflict(back)
		workspace, err := c.Workspace()
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
			view.Diagnostics(diags)
			return 1
		}
		sMgr, sDiags := back.StateMgr(workspace)
		if sDiags.HasErrors() {
			diags = diags.Append(fmt.Errorf("Error loading state: %s", sDiags.Err()))
			view.Diagnostics(diags)
			return 1
		}

		if err := sMgr.RefreshState(); err != nil {
			diags = diags.Append(fmt.Errorf("Error refreshing state: %s", err))
			view.Diagnostics(diags)
			return 1
		}

		state = sMgr.State()
	}

	if initArgs.Get {
		modsOutput, modsAbort, modsDiags := c.getModules(ctx, path, initArgs.TestsDirectory, rootModEarly, initArgs.Upgrade, view, policyClient)
		diags = diags.Append(modsDiags)
		if modsAbort || modsDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if modsOutput {
			// If we outputted information, then we need to output a newline
			// so that our success message is nicely spaced out from prior text.
			view.Output(views.EmptyMessage)
		}
	}

	// With all of the modules (hopefully) installed, we can now try to load the
	// whole configuration tree.
	config, confDiags := c.loadConfigWithTests(path, initArgs.TestsDirectory)
	// configDiags will be handled after the version constraint check, since an
	// incorrect version of terraform may produce errors for configuration
	// constructs added in later versions.

	// Before we go further, we'll check to make sure none of the modules in
	// the configuration declare that they don't support this Terraform
	// version, so we can produce a version-related error message rather than
	// potentially-confusing downstream errors.
	versionDiags := terraform.CheckCoreVersionRequirements(config)
	if versionDiags.HasErrors() {
		view.Diagnostics(versionDiags)
		return 1
	}

	// We've passed the core version check, now we can show any errors related to configuration
	// 1. Early errors from parsing the root module.
	// 2. Show any errors from initializing the backend.
	diags = diags.Append(earlyConfDiags)
	diags = diags.Append(backDiags)
	if earlyConfDiags.HasErrors() {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)))
		view.Diagnostics(diags)
		return 1
	}
	// If there are only backend errors, we won't show the InitConfigError preamble;
	// the config isn't the source of the errors it's probably the backend's own
	// Configure logic.
	if backDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// 3. Show any errors from loading the full configuration tree.
	diags = diags.Append(confDiags)
	if confDiags.HasErrors() {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)))
		view.Diagnostics(diags)
		return 1
	}

	if cb, ok := back.(*cloud.Cloud); ok {
		if c.RunningInAutomation {
			if err := cb.AssertImportCompatible(config); err != nil {
				diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Compatibility error", err.Error()))
				view.Diagnostics(diags)
				return 1
			}
		}
	}

	// Proceed with downloading providers
	var previousLocksWithPSSOverride *depsfile.Locks
	previousLocksWithPSSOverride = previousLocks.DeepCopy()
	if rootModEarly.StateStore != nil {
		// If a provider is used for state storage, the lock returned from getProvidersFromPSSConfig
		// is the only guaranteed source of that lock. We need to ensure its presence to influence
		// `getProviders`, else that method could download the PSS provider a second time, or download a different version.
		previousLocksWithPSSOverride = c.mergeLockedDependencies(pssLock, previousLocksWithPSSOverride)
	}
	stateProvidersOutput, finalLocks, stateProvidersDiags := c.getProviders(ctx, config, state, initArgs.Upgrade, previousLocksWithPSSOverride, initArgs.PluginPath, view, providerHook)
	diags = diags.Append(stateProvidersDiags)
	if stateProvidersDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if stateProvidersOutput {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// Update the dependency lock file, if it has changed.
	if rootModEarly.StateStore != nil && initArgs.Upgrade && !initArgs.Reconfigure {
		// If there's a provider upgrade happening (outside the context of -reconfigure),
		// then we override the state store provider lock with the pre-upgrade version.
		// Even if the upgrade process downloaded a newer version of the provider Terraform
		// will not use it due to the lock file being unchanged.
		finalLocks = c.mergeLockedDependencies(pssLock, finalLocks)
	}
	lockFileOutput, lockFileDiags := c.saveDependencyLockFile(previousLocks, finalLocks, c.incompleteProviders, initArgs.Lockfile, view)
	diags = diags.Append(lockFileDiags)
	if lockFileDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if lockFileOutput {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// If we accumulated any warnings along the way that weren't accompanied
	// by errors then we'll output them here so that the success message is
	// still the final thing shown.
	view.Diagnostics(diags)
	_, cloud := back.(*cloud.Cloud)
	output := views.OutputInitSuccessMessage
	if cloud {
		output = views.OutputInitSuccessCloudMessage
	}

	view.Output(output)

	if !c.RunningInAutomation {
		// If we're not running in an automation wrapper, give the user
		// some more detailed next steps that are appropriate for interactive
		// shell usage.
		output = views.OutputInitSuccessCLIMessage
		if cloud {
			output = views.OutputInitSuccessCLICloudMessage
		}
		view.Output(output)
	}
	return 0
}
