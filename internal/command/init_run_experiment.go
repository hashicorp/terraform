// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/didyoumean"
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
	// configDiags will be handled after:
	// - the version constraint check has happened
	// - and, the backend/state_store is initialised

	// Before we go further, we'll check to make sure none of the modules in
	// the configuration declare that they don't support this Terraform
	// version, so we can produce a version-related error message rather than
	// potentially-confusing downstream errors.
	versionDiags := terraform.CheckCoreVersionRequirements(config)
	if versionDiags.HasErrors() {
		view.Diagnostics(versionDiags)
		return 1
	}

	// We've passed the core version check, now we can show errors from the early configuration.
	// This prevents trying to initialise the backend with faulty configuration.
	if earlyConfDiags.HasErrors() {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)), earlyConfDiags)
		view.Diagnostics(diags)
		return 1
	}

	// Now the full configuration is loaded, we can download the providers specified in the configuration.
	// This is step one of a two-step provider download process
	// Providers may be downloaded by this code, but the dependency lock file is only updated later in `init`
	// after step two of provider download is complete.
	previousLocks, moreDiags := c.lockedDependencies()
	diags = diags.Append(moreDiags)

	configProvidersOutput, configLocks, configProviderDiags := c.getProvidersFromConfig(ctx, config, initArgs.Upgrade, initArgs.PluginPath, initArgs.Lockfile, view)
	diags = diags.Append(configProviderDiags)
	if configProviderDiags.HasErrors() {
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
		back, backendOutput, backDiags = c.initPssBackend(ctx, rootModEarly, initArgs, configLocks, view)
	default:
		// load the previously-stored backend config
		back, backDiags = c.Meta.backendFromState(ctx)
	}
	if backendOutput {
		header = true
	}
	if header {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// Show any errors from initializing the backend.
	// No preamble using `InitConfigError` is present, as we expect
	// any errors to from configuring the backend itself.
	diags = diags.Append(backDiags)
	if backDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// If everything is ok with the core version check and backend/state_store initialization,
	// show other errors from loading the full configuration tree.
	diags = diags.Append(confDiags)
	if confDiags.HasErrors() {
		diags = diags.Append(errors.New(view.PrepareMessage(views.InitConfigError)))
		view.Diagnostics(diags)
		return 1
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

	// Now the resource state is loaded, we can download the providers specified in the state but not the configuration.
	// This is step two of a two-step provider download process
	stateProvidersOutput, stateLocks, stateProvidersDiags := c.getProvidersFromState(ctx, state, configLocks, initArgs.Upgrade, initArgs.PluginPath, initArgs.Lockfile, view)
	diags = diags.Append(configProviderDiags)
	if stateProvidersDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if stateProvidersOutput {
		header = true
	}
	if header {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
	}

	// Now the two steps of provider download have happened, update the dependency lock file if it has changed.
	lockFileOutput, lockFileDiags := c.saveDependencyLockFile(previousLocks, configLocks, stateLocks, initArgs.Lockfile, view)
	diags = diags.Append(lockFileDiags)
	if lockFileDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if lockFileOutput {
		header = true
	}
	if header {
		// If we outputted information, then we need to output a newline
		// so that our success message is nicely spaced out from prior text.
		view.Output(views.EmptyMessage)
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

func (c *InitCommand) initPssBackend(ctx context.Context, root *configs.Module, initArgs *arguments.Init, configLocks *depsfile.Locks, view views.Init) (be backend.Backend, output bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "initialize backend")
	_ = ctx // prevent staticcheck from complaining to avoid a maintenance hazard of having the wrong ctx in scope here
	defer span.End()

	if root.StateStore != nil {
		view.Output(views.InitializingStateStoreMessage)
	} else {
		view.Output(views.InitializingBackendMessage)
	}

	var opts *BackendOpts
	switch {
	case root.StateStore != nil && root.Backend != nil:
		// We expect validation during config parsing to prevent mutually exclusive backend and state_store blocks,
		// but checking here just in case.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Conflicting backend and state_store configurations present during init",
			Detail: fmt.Sprintf("When initializing the backend there was configuration data present for both backend %q and state store %q. This is a bug in Terraform and should be reported.",
				root.Backend.Type,
				root.StateStore.Type,
			),
			Subject: &root.Backend.TypeRange,
		})
		return nil, true, diags
	case root.StateStore != nil:
		// state_store config present
		factory, fDiags := c.Meta.StateStoreProviderFactoryFromConfig(root.StateStore, configLocks)
		diags = diags.Append(fDiags)
		if fDiags.HasErrors() {
			return nil, true, diags
		}

		// If overrides supplied by -backend-config CLI flag, process them
		var configOverride hcl.Body
		if !initArgs.BackendConfig.Empty() {
			// We need to launch an instance of the provider to get the config of the state store for processing any overrides.
			provider, err := factory()
			defer provider.Close() // Stop the child process once we're done with it here.
			if err != nil {
				diags = diags.Append(fmt.Errorf("error when obtaining provider instance during state store initialization: %w", err))
				return nil, true, diags
			}

			resp := provider.GetProviderSchema()

			if len(resp.StateStores) == 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Provider does not support pluggable state storage",
					Detail: fmt.Sprintf("There are no state stores implemented by provider %s (%q)",
						root.StateStore.Provider.Name,
						root.StateStore.ProviderAddr),
					Subject: &root.StateStore.DeclRange,
				})
				return nil, true, diags
			}

			stateStoreSchema, exists := resp.StateStores[root.StateStore.Type]
			if !exists {
				suggestions := slices.Sorted(maps.Keys(resp.StateStores))
				suggestion := didyoumean.NameSuggestion(root.StateStore.Type, suggestions)
				if suggestion != "" {
					suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
				}
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "State store not implemented by the provider",
					Detail: fmt.Sprintf("State store %q is not implemented by provider %s (%q)%s",
						root.StateStore.Type, root.StateStore.Provider.Name,
						root.StateStore.ProviderAddr, suggestion),
					Subject: &root.StateStore.DeclRange,
				})
				return nil, true, diags
			}

			// Handle any overrides supplied via -backend-config CLI flags
			var overrideDiags tfdiags.Diagnostics
			configOverride, overrideDiags = c.backendConfigOverrideBody(initArgs.BackendConfig, stateStoreSchema.Body)
			diags = diags.Append(overrideDiags)
			if overrideDiags.HasErrors() {
				return nil, true, diags
			}
		}

		opts = &BackendOpts{
			StateStoreConfig:       root.StateStore,
			ProviderRequirements:   root.ProviderRequirements,
			Locks:                  configLocks,
			CreateDefaultWorkspace: initArgs.CreateDefaultWorkspace,
			ConfigOverride:         configOverride,
			Init:                   true,
			ViewType:               initArgs.ViewType,
		}

	case root.Backend != nil:
		// backend config present
		backendType := root.Backend.Type
		if backendType == "cloud" {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported backend type",
				Detail:   fmt.Sprintf("There is no explicit backend type named %q. To configure HCP Terraform, declare a 'cloud' block instead.", backendType),
				Subject:  &root.Backend.TypeRange,
			})
			return nil, true, diags
		}

		bf := backendInit.Backend(backendType)
		if bf == nil {
			detail := fmt.Sprintf("There is no backend type named %q.", backendType)
			if msg, removed := backendInit.RemovedBackends[backendType]; removed {
				detail = msg
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported backend type",
				Detail:   detail,
				Subject:  &root.Backend.TypeRange,
			})
			return nil, true, diags
		}

		b := bf()
		backendSchema := b.ConfigSchema()
		backendConfig := root.Backend

		// If overrides supplied by -backend-config CLI flag, process them
		var configOverride hcl.Body
		if !initArgs.BackendConfig.Empty() {
			var overrideDiags tfdiags.Diagnostics
			configOverride, overrideDiags = c.backendConfigOverrideBody(initArgs.BackendConfig, backendSchema)
			diags = diags.Append(overrideDiags)
			if overrideDiags.HasErrors() {
				return nil, true, diags
			}
		}

		opts = &BackendOpts{
			BackendConfig:  backendConfig,
			Locks:          configLocks,
			ConfigOverride: configOverride,
			Init:           true,
			ViewType:       initArgs.ViewType,
		}

	default:
		// No config; defaults to local state storage

		// If the user supplied a -backend-config on the CLI but no backend
		// block was found in the configuration, it's likely - but not
		// necessarily - a mistake. Return a warning.
		if !initArgs.BackendConfig.Empty() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Missing backend configuration",
				`-backend-config was used without a "backend" block in the configuration.

If you intended to override the default local backend configuration,
no action is required, but you may add an explicit backend block to your
configuration to clear this warning:

terraform {
  backend "local" {}
}

However, if you intended to override a defined backend, please verify that
the backend configuration is present and valid.
`,
			))
		}

		opts = &BackendOpts{
			Init:     true,
			Locks:    configLocks,
			ViewType: initArgs.ViewType,
		}
	}

	back, backDiags := c.Backend(opts)
	diags = diags.Append(backDiags)
	return back, true, diags
}
