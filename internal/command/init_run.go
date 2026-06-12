// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/plans"
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

	var policyClient policy.Client
	if len(initArgs.PolicyPaths) > 0 {
		var policyDiags policy.Diagnostics
		var stopClient func()
		policyClient, policyDiags, stopClient = c.PolicyClient(ctx, initArgs.PolicyPaths)
		defer stopClient()
		view.PolicyResults(nil, policyDiags)
		// if there has been any errors when setting up the policy client, we log them
		// and return early, as a failure to set up the policy client should terminate the init operation
		if policyDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
	}

	// If -state-provider-lock-file is set, we'll use that to obtain a new lock used for the state store provider
	// This will be 'upserted': it may be that the previous locks don't contain the provider being added. potentially due to being empty, or contain a different version.
	// The lock added will be used in the first step of provider download.
	//
	// We load locks from any pre-existing dependency lock file. These may or may not be altered by the -state-provider-lock-file flag.
	// The altered copy of the locks will be used to influence subsequent provider download steps.
	// The unaltered copy of the locks will be used at the end of the run to determine whether we need to update the dependency lock file on disk.
	previousLocks, locksDiags := c.lockedDependencies()
	diags = diags.Append(locksDiags)
	if locksDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	alteredPreviousLocks := previousLocks.DeepCopy()
	if initArgs.StateStoreProviderLockFile != "" {
		stateStoreLocks, lockDiags := depsfile.LoadLocksFromFile(initArgs.StateStoreProviderLockFile)
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
				"State store provider not found in -state-provider-lock-file dependency lock file",
				fmt.Sprintf("Terraform could not find the state store provider %q (%s) in the dependency lock file %q provided via the -state-provider-lock-file flag. Please ensure the lock file contains a lock for the state store provider and try again.",
					rootModEarly.StateStore.ProviderAddr.Type,
					rootModEarly.StateStore.ProviderAddr.ForDisplay(),
					initArgs.StateStoreProviderLockFile,
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

	policyResults := plans.NewPolicyResults()
	providerHook := &providerPolicyHook{
		client:        policyClient,
		policyResults: policyResults,
		rootModule:    rootModEarly,
	}

	var pssLocks *depsfile.Locks // May end up containing 0 or 1 lock.
	if rootModEarly.StateStore != nil {
		var configProvidersOutput bool
		var safeInitAction SafeInitAction
		var stateStoreProviderAuthResult *getproviders.PackageAuthenticationResult
		var configProviderDiags tfdiags.Diagnostics

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

		// Use alteredPreviousLocks, which may contain an additional lock supplied from the -state-provider-lock-file flag
		configProvidersOutput, pssLocks, safeInitAction, stateStoreProviderAuthResult, configProviderDiags = c.getProvidersFromPSSConfig(ctx, rootModEarly, alteredPreviousLocks, allowUpgrade, initArgs.PluginPath, initArgs.Lockfile, view)
		diags = diags.Append(configProviderDiags)
		if configProviderDiags.HasErrors() {
			view.PolicyResults(policyResults, nil)
			view.Diagnostics(diags)
			return 1
		}
		if configProvidersOutput {
			// If we outputted information, then we need to output a newline
			// so that our success message is nicely spaced out from prior text.
			view.Output(views.EmptyMessage)
		}

		// Course of action depends on the safeInitAction returned from getProvidersFromPSSConfig
		switch safeInitAction {
		case SafeInitActionProceed:
			// do nothing; provider is already trusted and there's no need to notify the user.
		case SafeInitActionRequireApproval:
			if c.input {
				// Prompt the user about trusting the provider used for state storage.
				diags = diags.Append(c.promptStateStorageProviderApproval(rootModEarly.StateStore.ProviderAddr, pssLocks, stateStoreProviderAuthResult))
				if diags.HasErrors() {
					view.Output(views.StateStoreProviderInteractiveRejectedMessage)
					view.Diagnostics(diags)
					return 1
				}
				view.Output(views.StateStoreProviderInteractiveApprovedMessage)
			} else {
				// Confirm that a lock was used to control download.
				// Note: we have to wait and do that here because at this point we know the provider was downloaded from a source that requires additional info about trust.
				if alteredPreviousLocks.Provider(rootModEarly.StateStore.ProviderAddr) == nil {
					// No lock was provided for the state store provider either through pre-existing locks or through the -state-provider-lock-file flag.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Missing lock for state store provider",
						"Terraform is initializing a state store for the first time in a non-interactive mode. In this scenario Terraform needs a pre-existing dependency lock for the state store provider to be present in the working directory's dependency lock file, or present in another file supplied via the -state-provider-lock-file flag. No lock was found for the state store provider. Please re-run the command using the -state-provider-lock-file flag.",
					))
					view.Diagnostics(diags)
					return 1
				}
				view.Output(views.StateStoreProviderAutomationApprovedMessage)
			}
		default:
			// Handle SafeInitActionInvalid or unexpected action types
			panic(fmt.Sprintf("When installing providers described in the config Terraform couldn't determine what 'safe init' action should be taken and returned action type %T. This is a bug in Terraform and should be reported.", safeInitAction))
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
		back, backendOutput, backDiags = c.initBackend(ctx, rootModEarly, initArgs, pssLocks, view)
	default:
		// load the previously-stored backend config
		back, backDiags = c.Meta.backendFromState(ctx)
	}
	if backendOutput {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
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
		modsOutput, modsAbort, policyResults, modsDiags := c.getModules(ctx, path, initArgs.TestsDirectory, rootModEarly, initArgs.Upgrade, view, policyClient)
		diags = diags.Append(modsDiags)
		view.PolicyResults(policyResults, nil)
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
	stateProvidersOutput, providerLocks, stateProvidersDiags := c.getProviders(ctx, config, state, initArgs.Upgrade, pssLocks, initArgs.PluginPath, view, providerHook)
	diags = diags.Append(stateProvidersDiags)
	if stateProvidersDiags.HasErrors() {
		view.PolicyResults(policyResults, nil)
		view.Diagnostics(diags)
		return 1
	}
	if stateProvidersOutput {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// Update the dependency lock file, if it has changed.
	lockFileOutput, lockFileDiags := c.saveDependencyLockFile(previousLocks, pssLocks, providerLocks, initArgs.Lockfile, view)
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
	view.PolicyResults(policyResults, nil)
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

// promptStateStorageProviderApproval is used when Terraform is unsure about the safety of the provider downloaded for state storage
// purposes, and we need to prompt the user to approve or reject using it.
func (c *InitCommand) promptStateStorageProviderApproval(stateStorageProvider addrs.Provider, configLocks *depsfile.Locks, authResult *getproviders.PackageAuthenticationResult) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// If we can receive input then we prompt for ok from the user
	lock := configLocks.Provider(stateStorageProvider)

	var hashList strings.Builder
	for _, hash := range lock.PreferredHashes() {
		hashList.WriteString(fmt.Sprintf("- %s\n", hash))
	}

	var authentication string
	if authResult != nil && authResult.KeyID != "" {
		authentication = fmt.Sprintf("%s, key ID %s", authResult.String(), authResult.KeyID)
	} else {
		authentication = authResult.String()
	}

	v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
		Id: "approve",
		Query: fmt.Sprintf(`Do you want to use provider %q (%s), version %s, for managing state?
Platform: %s
Authentication: %s
Hashes:
%s
`,
			lock.Provider().Type,
			lock.Provider(),
			lock.Version(),
			getproviders.CurrentPlatform.String(),
			authentication,
			hashList.String(),
		),
		Description: fmt.Sprintf(`Check the details above for provider %q and confirm that you trust the provider.
	Only 'yes' will be accepted to confirm.`, lock.Provider().Type),
	})
	if err != nil {
		return diags.Append(fmt.Errorf("Failed to approve use of state storage provider: %s", err))
	}
	if v != "yes" {
		return diags.Append(
			fmt.Errorf("State store provider %q (%s) was not approved, so init cannot continue.",
				lock.Provider().Type,
				lock.Provider(),
			),
		)
	}
	return diags
}
