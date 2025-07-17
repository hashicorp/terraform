// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// `runPssInit` is an altered version of the logic in `run` that contains changes
// related to the PSS project. This is used by the (InitCommand.Run method only if Terraform has
// experimental features enabled.
func (c *InitCommand) runPssInit(initArgs *arguments.Init, view views.Init) int {
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

	varArgs := initArgs.Vars.All()
	items := make([]arguments.FlagNameValue, len(varArgs))
	for i := range varArgs {
		items[i].Name = varArgs[i].Name
		items[i].Value = varArgs[i].Value
	}
	c.Meta.variableArgs = arguments.FlagNameValueSlice{Items: &items}

	// Copying the state only happens during backend migration, so setting
	// -force-copy implies -migrate-state
	if c.forceInitCopy {
		c.migrateState = true
	}

	if len(initArgs.PluginPath) > 0 {
		c.pluginPath = initArgs.PluginPath
	}

	// Validate the arg count and get the working directory
	path, err := ModulePath(initArgs.Args)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if err := c.storePluginPath(c.pluginPath); err != nil {
		diags = diags.Append(fmt.Errorf("Error saving -plugin-dir to workspace directory: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Initialization can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	// This will track whether we outputted anything so that we know whether
	// to output a newline before the success message
	var header bool

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
		header = true

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

	if initArgs.Get {
		modsOutput, modsAbort, modsDiags := c.getModules(ctx, path, initArgs.TestsDirectory, rootModEarly, initArgs.Upgrade, view)
		diags = diags.Append(modsDiags)
		if modsAbort || modsDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
		if modsOutput {
			header = true
		}
	}

	// With all of the modules (hopefully) installed, we can now try to load the
	// whole configuration tree.
	config, confDiags := c.loadConfigWithTests(path, initArgs.TestsDirectory)
	// configDiags will be handled after the version constraint check, since an
	// incorrect version of terraform may be producing errors for configuration
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

	// Now the full configuration is loaded, we can download the providers specified in the configuration.
	// This is step one of a two-step provider download process
	// Providers may be downloaded by this code, but the dependency lock file is only updated later in `init`
	// after step two of provider download is complete.
	configProvidersOutput, configProvidersOutcome, configLocks, configProviderDiags := c.getProvidersFromConfig(ctx, config, initArgs.Upgrade, initArgs.PluginPath, initArgs.Lockfile, view)
	diags = diags.Append(configProviderDiags)
	if configProvidersOutcome == ProviderDownloadAborted || configProviderDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if configProvidersOutput {
		header = true
	}

	// If we outputted information, then we need to output a newline
	// so that our success message is nicely spaced out from prior text.
	if header {
		view.Output(views.EmptyMessage)
	}

	var back backend.Backend

	var backDiags tfdiags.Diagnostics
	var backendOutput bool
	switch {
	case initArgs.Cloud && rootModEarly.CloudConfig != nil:
		back, backendOutput, backDiags = c.initCloud(ctx, rootModEarly, initArgs.BackendConfig, initArgs.ViewType, view)
	case initArgs.Backend:
		// TODO(SarahFrench/radeksimko) - pass information about config locks into initBackend to
		// enable PSS
		back, backendOutput, backDiags = c.initBackend(ctx, rootModEarly, initArgs.BackendConfig, initArgs.ViewType, view)
	default:
		// load the previously-stored backend config
		back, backDiags = c.Meta.backendFromState(ctx)
	}
	if backendOutput {
		header = true
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
		sMgr, err := back.StateMgr(workspace)
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error loading state: %s", err))
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

	// Now the resource state is loaded, we can download the providers specified in the state but not the configuration.
	// This is step two of a two-step provider download process
	stateProvidersOutput, stateProvidersOutcome, stateLocks, stateProvidersDiags := c.getProvidersFromState(ctx, state, initArgs.Upgrade, initArgs.PluginPath, initArgs.Lockfile, view)
	diags = diags.Append(configProviderDiags)
	if stateProvidersOutcome == ProviderDownloadAborted || stateProvidersDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if stateProvidersOutput {
		header = true
	}
	if configProvidersOutcome == ProviderDownloadLocksChanged ||
		stateProvidersOutcome == ProviderDownloadLocksChanged {
		// Only update the dependency lock file if locks differ.
		// We update the lock file once, after all dependencies are collected,
		// to avoid scenarios where Terraform is interrupted between partial updates.
		merged := c.mergeLockedDependencies(stateLocks, configLocks)
		lockFileDiags := c.replaceLockedDependencies(merged)
		diags = diags.Append(lockFileDiags)
	}

	// If we outputted information, then we need to output a newline
	// so that our success message is nicely spaced out from prior text.
	if header {
		view.Output(views.EmptyMessage)
	}

	// As Terraform version-related diagnostics are handled above, we can now
	// check the diagnostics from the early configuration and the backend.
	diags = diags.Append(earlyConfDiags)
	diags = diags.Append(backDiags)
	if earlyConfDiags.HasErrors() {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)))
		view.Diagnostics(diags)
		return 1
	}

	// Now, we can show any errors from initializing the backend, but we won't
	// show the InitConfigError preamble as we didn't detect problems with
	// the early configuration.
	if backDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// If everything is ok with the core version check and backend initialization,
	// show other errors from loading the full configuration tree.
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
