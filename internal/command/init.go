// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/posener/complete"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	var diags tfdiags.Diagnostics
	args = c.Meta.process(args)
	initArgs, initDiags := arguments.ParseInit(args)

	view := views.NewInit(initArgs.ViewType, c.View)

	if initDiags.HasErrors() {
		diags = diags.Append(initDiags)
		view.Diagnostics(diags)
		return 1
	}

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

		empty, err := configs.IsEmptyDir(path)
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
	empty, err := configs.IsEmptyDir(path)
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
		diags = diags.Append(fmt.Errorf(view.PrepareMessage(views.InitConfigError)), earlyConfDiags)
		view.Diagnostics(diags)

		return 1
	}

	var back backend.Backend

	// There may be config errors or backend init errors but these will be shown later _after_
	// checking for core version requirement errors.
	var backDiags tfdiags.Diagnostics
	var backendOutput bool

	switch {
	case initArgs.Cloud && rootModEarly.CloudConfig != nil:
		back, backendOutput, backDiags = c.initCloud(ctx, rootModEarly, initArgs.BackendConfig, initArgs.ViewType, view)
	case initArgs.Backend:
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

	// We've passed the core version check, now we can show errors from the
	// configuration and backend initialisation.

	// Now, we can check the diagnostics from the early configuration and the
	// backend.
	diags = diags.Append(earlyConfDiags)
	diags = diags.Append(backDiags)
	if earlyConfDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf(view.PrepareMessage(views.InitConfigError)))
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
		diags = diags.Append(fmt.Errorf(view.PrepareMessage(views.InitConfigError)))
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

	// Now that we have loaded all modules, check the module tree for missing providers.
	providersOutput, providersAbort, providerDiags := c.getProviders(ctx, config, state, initArgs.Upgrade, initArgs.PluginPath, initArgs.Lockfile, view)
	diags = diags.Append(providerDiags)
	if providersAbort || providerDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}
	if providersOutput {
		header = true
	}

	// If we outputted information, then we need to output a newline
	// so that our success message is nicely spaced out from prior text.
	if header {
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

func (c *InitCommand) getModules(ctx context.Context, path, testsDir string, earlyRoot *configs.Module, upgrade bool, view views.Init) (output bool, abort bool, diags tfdiags.Diagnostics) {
	testModules := false // We can also have modules buried in test files.
	for _, file := range earlyRoot.Tests {
		for _, run := range file.Runs {
			if run.Module != nil {
				testModules = true
			}
		}
	}

	if len(earlyRoot.ModuleCalls) == 0 && !testModules {
		// Nothing to do
		return false, false, nil
	}

	ctx, span := tracer.Start(ctx, "install modules", trace.WithAttributes(
		attribute.Bool("upgrade", upgrade),
	))
	defer span.End()

	if upgrade {
		view.Output(views.UpgradingModulesMessage)
	} else {
		view.Output(views.InitializingModulesMessage)
	}

	hooks := uiModuleInstallHooks{
		Ui:             c.Ui,
		ShowLocalPaths: true,
		View:           view,
	}

	installAbort, installDiags := c.installModules(ctx, path, testsDir, upgrade, false, hooks)
	diags = diags.Append(installDiags)

	// At this point, installModules may have generated error diags or been
	// aborted by SIGINT. In any case we continue and the manifest as best
	// we can.

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if c.configLoader != nil {
		if err := c.configLoader.RefreshModules(); err != nil {
			// Should never happen
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read module manifest",
				fmt.Sprintf("After installing modules, Terraform could not re-read the manifest of installed modules. This is a bug in Terraform. %s.", err),
			))
		}
	}

	return true, installAbort, diags
}

func (c *InitCommand) initCloud(ctx context.Context, root *configs.Module, extraConfig arguments.FlagNameValueSlice, viewType arguments.ViewType, view views.Init) (be backend.Backend, output bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "initialize HCP Terraform")
	_ = ctx // prevent staticcheck from complaining to avoid a maintenence hazard of having the wrong ctx in scope here
	defer span.End()

	view.Output(views.InitializingTerraformCloudMessage)

	if len(extraConfig.AllItems()) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid command-line option",
			"The -backend-config=... command line option is only for state backends, and is not applicable to HCP Terraform-based configurations.\n\nTo change the set of workspaces associated with this configuration, edit the Cloud configuration block in the root module.",
		))
		return nil, true, diags
	}

	backendConfig := root.CloudConfig.ToBackendConfig()

	opts := &BackendOpts{
		Config:   &backendConfig,
		Init:     true,
		ViewType: viewType,
	}

	back, backDiags := c.Backend(opts)
	diags = diags.Append(backDiags)
	return back, true, diags
}

func (c *InitCommand) initBackend(ctx context.Context, root *configs.Module, extraConfig arguments.FlagNameValueSlice, viewType arguments.ViewType, view views.Init) (be backend.Backend, output bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "initialize backend")
	_ = ctx // prevent staticcheck from complaining to avoid a maintenence hazard of having the wrong ctx in scope here
	defer span.End()

	view.Output(views.InitializingBackendMessage)

	var backendConfig *configs.Backend
	var backendConfigOverride hcl.Body
	if root.Backend != nil {
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
		backendConfig = root.Backend

		var overrideDiags tfdiags.Diagnostics
		backendConfigOverride, overrideDiags = c.backendConfigOverrideBody(extraConfig, backendSchema)
		diags = diags.Append(overrideDiags)
		if overrideDiags.HasErrors() {
			return nil, true, diags
		}
	} else {
		// If the user supplied a -backend-config on the CLI but no backend
		// block was found in the configuration, it's likely - but not
		// necessarily - a mistake. Return a warning.
		if !extraConfig.Empty() {
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
	}

	opts := &BackendOpts{
		Config:         backendConfig,
		ConfigOverride: backendConfigOverride,
		Init:           true,
		ViewType:       viewType,
	}

	back, backDiags := c.Backend(opts)
	diags = diags.Append(backDiags)
	return back, true, diags
}

// Load the complete module tree, and fetch any missing providers.
// This method outputs its own Ui.
func (c *InitCommand) getProviders(ctx context.Context, config *configs.Config, state *states.State, upgrade bool, pluginDirs []string, flagLockfile string, view views.Init) (output, abort bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "install providers")
	defer span.End()

	// Dev overrides cause the result of "terraform init" to be irrelevant for
	// any overridden providers, so we'll warn about it to avoid later
	// confusion when Terraform ends up using a different provider than the
	// lock file called for.
	diags = diags.Append(c.providerDevOverrideInitWarnings())

	// First we'll collect all the provider dependencies we can see in the
	// configuration and the state.
	reqs, hclDiags := config.ProviderRequirements()
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return false, true, diags
	}
	if state != nil {
		stateReqs := state.ProviderRequirements()
		reqs = reqs.Merge(stateReqs)
	}

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

	previousLocks, moreDiags := c.lockedDependencies()
	diags = diags.Append(moreDiags)

	if diags.HasErrors() {
		return false, true, diags
	}

	var inst *providercache.Installer
	if len(pluginDirs) == 0 {
		// By default we use a source that looks for providers in all of the
		// standard locations, possibly customized by the user in CLI config.
		inst = c.providerInstaller()
	} else {
		// If the user passes at least one -plugin-dir then that circumvents
		// the usual sources and forces Terraform to consult only the given
		// directories. Anything not available in one of those directories
		// is not available for installation.
		source := c.providerCustomLocalDirectorySource(pluginDirs)
		inst = c.providerInstallerCustomSource(source)

		// The default (or configured) search paths are logged earlier, in provider_source.go
		// Log that those are being overridden by the `-plugin-dir` command line options
		log.Println("[DEBUG] init: overriding provider plugin search paths")
		log.Printf("[DEBUG] will search for provider plugins in %s", pluginDirs)
	}

	// We want to print out a nice warning if we don't manage to pull
	// checksums for all our providers. This is tracked via callbacks
	// and incomplete providers are stored here for later analysis.
	var incompleteProviders []string

	// Because we're currently just streaming a series of events sequentially
	// into the terminal, we're showing only a subset of the events to keep
	// things relatively concise. Later it'd be nice to have a progress UI
	// where statuses update in-place, but we can't do that as long as we
	// are shimming our vt100 output to the legacy console API on Windows.
	evts := &providercache.InstallerEvents{
		PendingProviders: func(reqs map[addrs.Provider]getproviders.VersionConstraints) {
			view.Output(views.InitializingProviderPluginMessage)
		},
		ProviderAlreadyInstalled: func(provider addrs.Provider, selectedVersion getproviders.Version) {
			view.LogInitMessage(views.ProviderAlreadyInstalledMessage, provider.ForDisplay(), selectedVersion)
		},
		BuiltInProviderAvailable: func(provider addrs.Provider) {
			view.LogInitMessage(views.BuiltInProviderAvailableMessage, provider.ForDisplay())
		},
		BuiltInProviderFailure: func(provider addrs.Provider, err error) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid dependency on built-in provider",
				fmt.Sprintf("Cannot use %s: %s.", provider.ForDisplay(), err),
			))
		},
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
		LinkFromCacheBegin: func(provider addrs.Provider, version getproviders.Version, cacheRoot string) {
			view.LogInitMessage(views.UsingProviderFromCacheDirInfo, provider.ForDisplay(), version)
		},
		FetchPackageBegin: func(provider addrs.Provider, version getproviders.Version, location getproviders.PackageLocation) {
			view.LogInitMessage(views.InstallingProviderMessage, provider.ForDisplay(), version)
		},
		QueryPackagesFailure: func(provider addrs.Provider, err error) {
			switch errorTy := err.(type) {
			case getproviders.ErrProviderNotFound:
				sources := errorTy.Sources
				displaySources := make([]string, len(sources))
				for i, source := range sources {
					displaySources[i] = fmt.Sprintf("  - %s", source)
				}
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to query available provider packages",
					fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s\n\n%s",
						provider.ForDisplay(), err, strings.Join(displaySources, "\n"),
					),
				))
			case getproviders.ErrRegistryProviderNotKnown:
				// We might be able to suggest an alternative provider to use
				// instead of this one.
				suggestion := fmt.Sprintf("\n\nAll modules should specify their required_providers so that external consumers will get the correct providers when using a module. To see which modules are currently depending on %s, run the following command:\n    terraform providers", provider.ForDisplay())
				alternative := getproviders.MissingProviderSuggestion(ctx, provider, inst.ProviderSource(), reqs)
				if alternative != provider {
					suggestion = fmt.Sprintf(
						"\n\nDid you intend to use %s? If so, you must specify that source address in each module which requires that provider. To see which modules are currently depending on %s, run the following command:\n    terraform providers",
						alternative.ForDisplay(), provider.ForDisplay(),
					)
				}

				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to query available provider packages",
					fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s%s",
						provider.ForDisplay(), err, suggestion,
					),
				))
			case getproviders.ErrHostNoProviders:
				switch {
				case errorTy.Hostname == svchost.Hostname("github.com") && !errorTy.HasOtherVersion:
					// If a user copies the URL of a GitHub repository into
					// the source argument and removes the schema to make it
					// provider-address-shaped then that's one way we can end up
					// here. We'll use a specialized error message in anticipation
					// of that mistake. We only do this if github.com isn't a
					// provider registry, to allow for the (admittedly currently
					// rather unlikely) possibility that github.com starts being
					// a real Terraform provider registry in the future.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider registry host",
						fmt.Sprintf("The given source address %q specifies a GitHub repository rather than a Terraform provider. Refer to the documentation of the provider to find the correct source address to use.",
							provider.String(),
						),
					))

				case errorTy.HasOtherVersion:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider registry host",
						fmt.Sprintf("The host %q given in provider source address %q does not offer a Terraform provider registry that is compatible with this Terraform version, but it may be compatible with a different Terraform version.",
							errorTy.Hostname, provider.String(),
						),
					))

				default:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider registry host",
						fmt.Sprintf("The host %q given in provider source address %q does not offer a Terraform provider registry.",
							errorTy.Hostname, provider.String(),
						),
					))
				}

			case getproviders.ErrRequestCanceled:
				// We don't attribute cancellation to any particular operation,
				// but rather just emit a single general message about it at
				// the end, by checking ctx.Err().

			default:
				suggestion := fmt.Sprintf("\n\nTo see which modules are currently depending on %s and what versions are specified, run the following command:\n    terraform providers", provider.ForDisplay())
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to query available provider packages",
					fmt.Sprintf("Could not retrieve the list of available versions for provider %s: %s%s",
						provider.ForDisplay(), err, suggestion,
					),
				))
			}

		},
		QueryPackagesWarning: func(provider addrs.Provider, warnings []string) {
			displayWarnings := make([]string, len(warnings))
			for i, warning := range warnings {
				displayWarnings[i] = fmt.Sprintf("- %s", warning)
			}

			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Additional provider information from registry",
				fmt.Sprintf("The remote registry returned warnings for %s:\n%s",
					provider.String(),
					strings.Join(displayWarnings, "\n"),
				),
			))
		},
		LinkFromCacheFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to install provider from shared cache",
				fmt.Sprintf("Error while importing %s v%s from the shared cache directory: %s.", provider.ForDisplay(), version, err),
			))
		},
		FetchPackageFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			const summaryIncompatible = "Incompatible provider version"
			switch err := err.(type) {
			case getproviders.ErrProtocolNotSupported:
				closestAvailable := err.Suggestion
				switch {
				case closestAvailable == getproviders.UnspecifiedVersion:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						summaryIncompatible,
						fmt.Sprintf(errProviderVersionIncompatible, provider.String()),
					))
				case version.GreaterThan(closestAvailable):
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						summaryIncompatible,
						fmt.Sprintf(providerProtocolTooNew, provider.ForDisplay(),
							version, tfversion.String(), closestAvailable, closestAvailable,
							getproviders.VersionConstraintsString(reqs[provider]),
						),
					))
				default: // version is less than closestAvailable
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						summaryIncompatible,
						fmt.Sprintf(providerProtocolTooOld, provider.ForDisplay(),
							version, tfversion.String(), closestAvailable, closestAvailable,
							getproviders.VersionConstraintsString(reqs[provider]),
						),
					))
				}
			case getproviders.ErrPlatformNotSupported:
				switch {
				case err.MirrorURL != nil:
					// If we're installing from a mirror then it may just be
					// the mirror lacking the package, rather than it being
					// unavailable from upstream.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						summaryIncompatible,
						fmt.Sprintf(
							"Your chosen provider mirror at %s does not have a %s v%s package available for your current platform, %s.\n\nProvider releases are separate from Terraform CLI releases, so this provider might not support your current platform. Alternatively, the mirror itself might have only a subset of the plugin packages available in the origin registry, at %s.",
							err.MirrorURL, err.Provider, err.Version, err.Platform,
							err.Provider.Hostname,
						),
					))
				default:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						summaryIncompatible,
						fmt.Sprintf(
							"Provider %s v%s does not have a package available for your current platform, %s.\n\nProvider releases are separate from Terraform CLI releases, so not all providers are available for all platforms. Other versions of this provider may have different platforms supported.",
							err.Provider, err.Version, err.Platform,
						),
					))
				}

			case getproviders.ErrRequestCanceled:
				// We don't attribute cancellation to any particular operation,
				// but rather just emit a single general message about it at
				// the end, by checking ctx.Err().

			default:
				// We can potentially end up in here under cancellation too,
				// in spite of our getproviders.ErrRequestCanceled case above,
				// because not all of the outgoing requests we do under the
				// "fetch package" banner are source metadata requests.
				// In that case we will emit a redundant error here about
				// the request being cancelled, but we'll still detect it
				// as a cancellation after the installer returns and do the
				// normal cancellation handling.

				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to install provider",
					fmt.Sprintf("Error while installing %s v%s: %s", provider.ForDisplay(), version, err),
				))
			}
		},
		FetchPackageSuccess: func(provider addrs.Provider, version getproviders.Version, localDir string, authResult *getproviders.PackageAuthenticationResult) {
			var keyID string
			if authResult != nil && authResult.ThirdPartySigned() {
				keyID = authResult.KeyID
			}
			if keyID != "" {
				keyID = view.PrepareMessage(views.KeyID, keyID)
			}

			view.LogInitMessage(views.InstalledProviderVersionInfo, provider.ForDisplay(), version, authResult, keyID)
		},
		ProvidersLockUpdated: func(provider addrs.Provider, version getproviders.Version, localHashes []getproviders.Hash, signedHashes []getproviders.Hash, priorHashes []getproviders.Hash) {
			// We're going to use this opportunity to track if we have any
			// "incomplete" installs of providers. An incomplete install is
			// when we are only going to write the local hashes into our lock
			// file which means a `terraform init` command will fail in future
			// when used on machines of a different architecture.
			//
			// We want to print a warning about this.

			if len(signedHashes) > 0 {
				// If we have any signedHashes hashes then we don't worry - as
				// we know we retrieved all available hashes for this version
				// anyway.
				return
			}

			// If local hashes and prior hashes are exactly the same then
			// it means we didn't record any signed hashes previously, and
			// we know we're not adding any extra in now (because we already
			// checked the signedHashes), so that's a problem.
			//
			// In the actual check here, if we have any priorHashes and those
			// hashes are not the same as the local hashes then we're going to
			// accept that this provider has been configured correctly.
			if len(priorHashes) > 0 && !reflect.DeepEqual(localHashes, priorHashes) {
				return
			}

			// Now, either signedHashes is empty, or priorHashes is exactly the
			// same as our localHashes which means we never retrieved the
			// signedHashes previously.
			//
			// Either way, this is bad. Let's complain/warn.
			incompleteProviders = append(incompleteProviders, provider.ForDisplay())
		},
		ProvidersFetched: func(authResults map[addrs.Provider]*getproviders.PackageAuthenticationResult) {
			thirdPartySigned := false
			for _, authResult := range authResults {
				if authResult.ThirdPartySigned() {
					thirdPartySigned = true
					break
				}
			}
			if thirdPartySigned {
				view.LogInitMessage(views.PartnerAndCommunityProvidersMessage)
			}
		},
	}
	ctx = evts.OnContext(ctx)

	mode := providercache.InstallNewProvidersOnly
	if upgrade {
		if flagLockfile == "readonly" {
			diags = diags.Append(fmt.Errorf("The -upgrade flag conflicts with -lockfile=readonly."))
			view.Diagnostics(diags)
			return true, true, diags
		}

		mode = providercache.InstallUpgrades
	}
	newLocks, err := inst.EnsureProviderVersions(ctx, previousLocks, reqs, mode)
	if ctx.Err() == context.Canceled {
		diags = diags.Append(fmt.Errorf("Provider installation was canceled by an interrupt signal."))
		view.Diagnostics(diags)
		return true, true, diags
	}
	if err != nil {
		// The errors captured in "err" should be redundant with what we
		// received via the InstallerEvents callbacks above, so we'll
		// just return those as long as we have some.
		if !diags.HasErrors() {
			diags = diags.Append(err)
		}

		return true, true, diags
	}

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
			// check if required provider dependences change
			if !newLocks.EqualProviderAddress(previousLocks) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					`Provider dependency changes detected`,
					`Changes to the required provider dependencies were detected, but the lock file is read-only. To use and record these requirements, run "terraform init" without the "-lockfile=readonly" flag.`,
				))
				return true, true, diags
			}

			// suppress updating the file to record any new information it learned,
			// such as a hash using a new scheme.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				`Provider lock file not updated`,
				`Changes to the provider selections were detected, but not saved in the .terraform.lock.hcl file. To record these selections, run "terraform init" without the "-lockfile=readonly" flag.`,
			))
			return true, false, diags
		}

		// Jump in here and add a warning if any of the providers are incomplete.
		if len(incompleteProviders) > 0 && !inst.GlobalCacheDirMayBreakDependencyLockFile() {
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
			view.Output(views.LockInfo)
		} else {
			view.Output(views.DependenciesLockChangesInfo)
		}

		moreDiags = c.replaceLockedDependencies(newLocks)
		diags = diags.Append(moreDiags)
	}

	return true, false, diags
}

// backendConfigOverrideBody interprets the raw values of -backend-config
// arguments into a hcl Body that should override the backend settings given
// in the configuration.
//
// If the result is nil then no override needs to be provided.
//
// If the returned diagnostics contains errors then the returned body may be
// incomplete or invalid.
func (c *InitCommand) backendConfigOverrideBody(flags arguments.FlagNameValueSlice, schema *configschema.Block) (hcl.Body, tfdiags.Diagnostics) {
	items := flags.AllItems()
	if len(items) == 0 {
		return nil, nil
	}

	var ret hcl.Body
	var diags tfdiags.Diagnostics
	synthVals := make(map[string]cty.Value)

	mergeBody := func(newBody hcl.Body) {
		if ret == nil {
			ret = newBody
		} else {
			ret = configs.MergeBodies(ret, newBody)
		}
	}
	flushVals := func() {
		if len(synthVals) == 0 {
			return
		}
		newBody := configs.SynthBody("-backend-config=...", synthVals)
		mergeBody(newBody)
		synthVals = make(map[string]cty.Value)
	}

	if len(items) == 1 && items[0].Value == "" {
		// Explicitly remove all -backend-config options.
		// We do this by setting an empty but non-nil ConfigOverrides.
		return configs.SynthBody("-backend-config=''", synthVals), diags
	}

	for _, item := range items {
		eq := strings.Index(item.Value, "=")

		if eq == -1 {
			// The value is interpreted as a filename.
			newBody, fileDiags := c.loadHCLFile(item.Value)
			diags = diags.Append(fileDiags)
			if fileDiags.HasErrors() {
				continue
			}
			// Generate an HCL body schema for the backend block.
			var bodySchema hcl.BodySchema
			for name := range schema.Attributes {
				// We intentionally ignore the `Required` attribute here
				// because backend config override files can be partial. The
				// goal is to make sure we're not loading a file with
				// extraneous attributes or blocks.
				bodySchema.Attributes = append(bodySchema.Attributes, hcl.AttributeSchema{
					Name: name,
				})
			}
			for name, block := range schema.BlockTypes {
				var labelNames []string
				if block.Nesting == configschema.NestingMap {
					labelNames = append(labelNames, "key")
				}
				bodySchema.Blocks = append(bodySchema.Blocks, hcl.BlockHeaderSchema{
					Type:       name,
					LabelNames: labelNames,
				})
			}
			// Verify that the file body matches the expected backend schema.
			_, schemaDiags := newBody.Content(&bodySchema)
			diags = diags.Append(schemaDiags)
			if schemaDiags.HasErrors() {
				continue
			}
			flushVals() // deal with any accumulated individual values first
			mergeBody(newBody)
		} else {
			name := item.Value[:eq]
			rawValue := item.Value[eq+1:]
			attrS := schema.Attributes[name]
			if attrS == nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid backend configuration argument",
					fmt.Sprintf("The backend configuration argument %q given on the command line is not expected for the selected backend type.", name),
				))
				continue
			}
			value, valueDiags := configValueFromCLI(item.String(), rawValue, attrS.Type)
			diags = diags.Append(valueDiags)
			if valueDiags.HasErrors() {
				continue
			}
			synthVals[name] = value
		}
	}

	flushVals()

	return ret, diags
}

func (c *InitCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("")
}

func (c *InitCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-backend":        completePredictBoolean,
		"-cloud":          completePredictBoolean,
		"-backend-config": complete.PredictFiles("*.tfvars"), // can also be key=value, but we can't "predict" that
		"-force-copy":     complete.PredictNothing,
		"-from-module":    completePredictModuleSource,
		"-get":            completePredictBoolean,
		"-input":          completePredictBoolean,
		"-lock":           completePredictBoolean,
		"-lock-timeout":   complete.PredictAnything,
		"-no-color":       complete.PredictNothing,
		"-json":           complete.PredictNothing,
		"-plugin-dir":     complete.PredictDirs(""),
		"-reconfigure":    complete.PredictNothing,
		"-migrate-state":  complete.PredictNothing,
		"-upgrade":        completePredictBoolean,
	}
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: terraform [global options] init [options]

  Initialize a new or existing Terraform working directory by creating
  initial files, loading any remote state, downloading modules, etc.

  This is the first command that should be run for any new or existing
  Terraform configuration per machine. This sets up all the local data
  necessary to run Terraform that is typically not committed to version
  control.

  This command is always safe to run multiple times. Though subsequent runs
  may give errors, this command will never delete your configuration or
  state. Even so, if you have important information, please back it up prior
  to running this command, just in case.

Options:

  -backend=false          Disable backend or HCP Terraform initialization
                          for this configuration and use what was previously
                          initialized instead.

                          aliases: -cloud=false

  -backend-config=path    Configuration to be merged with what is in the
                          configuration file's 'backend' block. This can be
                          either a path to an HCL file with key/value
                          assignments (same format as terraform.tfvars) or a
                          'key=value' format, and can be specified multiple
                          times. The backend type must be in the configuration
                          itself.

  -force-copy             Suppress prompts about copying state data when
                          initializating a new state backend. This is
                          equivalent to providing a "yes" to all confirmation
                          prompts.

  -from-module=SOURCE     Copy the contents of the given module into the target
                          directory before initialization.

  -get=false              Disable downloading modules for this configuration.

  -input=false            Disable interactive prompts. Note that some actions may
                          require interactive prompts and will error if input is
                          disabled.

  -lock=false             Don't hold a state lock during backend migration.
                          This is dangerous if others might concurrently run
                          commands against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -no-color               If specified, output won't contain any color.

  -json                   If specified, machine readable output will be
                          printed in JSON format.

  -plugin-dir             Directory containing plugin binaries. This overrides all
                          default search paths for plugins, and prevents the
                          automatic installation of plugins. This flag can be used
                          multiple times.

  -reconfigure            Reconfigure a backend, ignoring any saved
                          configuration.

  -migrate-state          Reconfigure a backend, and attempt to migrate any
                          existing state.

  -upgrade                Install the latest module and provider versions
                          allowed within configured constraints, overriding the
                          default behavior of selecting exactly the version
                          recorded in the dependency lockfile.

  -lockfile=MODE          Set a dependency lockfile mode.
                          Currently only "readonly" is valid.

  -ignore-remote-version  A rare option used for HCP Terraform and the remote backend
                          only. Set this to ignore checking that the local and remote
                          Terraform versions use compatible state representations, making
                          an operation proceed even when there is a potential mismatch.
                          See the documentation on configuring Terraform with
                          HCP Terraform or Terraform Enterprise for more information.

  -test-directory=path    Set the Terraform test directory, defaults to "tests".

`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Prepare your working directory for other commands"
}

const errInitCopyNotEmpty = `
The working directory already contains files. The -from-module option requires
an empty directory into which a copy of the referenced module will be placed.

To initialize the configuration already in this working directory, omit the
-from-module option.
`

// providerProtocolTooOld is a message sent to the CLI UI if the provider's
// supported protocol versions are too old for the user's version of terraform,
// but a newer version of the provider is compatible.
const providerProtocolTooOld = `Provider %q v%s is not compatible with Terraform %s.
Provider version %s is the latest compatible version. Select it with the following version constraint:
	version = %q

Terraform checked all of the plugin versions matching the given constraint:
	%s

Consult the documentation for this provider for more information on compatibility between provider and Terraform versions.
`

// providerProtocolTooNew is a message sent to the CLI UI if the provider's
// supported protocol versions are too new for the user's version of terraform,
// and the user could either upgrade terraform or choose an older version of the
// provider.
const providerProtocolTooNew = `Provider %q v%s is not compatible with Terraform %s.
You need to downgrade to v%s or earlier. Select it with the following constraint:
	version = %q

Terraform checked all of the plugin versions matching the given constraint:
	%s

Consult the documentation for this provider for more information on compatibility between provider and Terraform versions.
Alternatively, upgrade to the latest version of Terraform for compatibility with newer provider releases.
`

// No version of the provider is compatible.
const errProviderVersionIncompatible = `No compatible versions of provider %s were found.`

// incompleteLockFileInformationHeader is the summary displayed to users when
// the lock file has only recorded local hashes.
const incompleteLockFileInformationHeader = `Incomplete lock file information for providers`

// incompleteLockFileInformationBody is the body of text displayed to users when
// the lock file has only recorded local hashes.
const incompleteLockFileInformationBody = `Due to your customized provider installation methods, Terraform was forced to calculate lock file checksums locally for the following providers:
  - %s

The current .terraform.lock.hcl file only includes checksums for %s, so Terraform running on another platform will fail to install these providers.

To calculate additional checksums for another platform, run:
  terraform providers lock -platform=linux_amd64
(where linux_amd64 is the platform to generate)`
