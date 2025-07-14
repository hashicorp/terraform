// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"log"
	"maps"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/posener/complete"
	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta

	incompleteProviders []string
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

	// The else condition below invokes the original logic of the init command.
	// An experimental version of the init code will be used if:
	// 	> The user uses an experimental version of TF (alpha or built from source)
	//  > Either the flag -enable-pluggable-state-storage-experiment is passed to the init command.
	//  > Or, the environment variable TF_ENABLE_PLUGGABLE_STATE_STORAGE is set to any value.
	if v := os.Getenv("TF_ENABLE_PLUGGABLE_STATE_STORAGE"); v != "" {
		initArgs.EnablePssExperiment = true
	}
	if c.Meta.AllowExperimentalFeatures && initArgs.EnablePssExperiment {
		// TODO(SarahFrench/radeksimko): Remove forked init logic once feature is no longer experimental
		return c.runPssInit(initArgs, view)
	} else {
		return c.run(initArgs, view)
	}
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
	_ = ctx // prevent staticcheck from complaining to avoid a maintenance hazard of having the wrong ctx in scope here
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
		BackendConfig: &backendConfig,
		Init:          true,
		ViewType:      viewType,
	}

	back, backDiags := c.Backend(opts)
	diags = diags.Append(backDiags)
	return back, true, diags
}

func (c *InitCommand) initBackend(ctx context.Context, root *configs.Module, extraConfig arguments.FlagNameValueSlice, viewType arguments.ViewType, configLocks *depsfile.Locks, view views.Init) (be backend.Backend, output bool, diags tfdiags.Diagnostics) {
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
		factory, fDiags := c.Meta.GetStateStoreProviderFactory(root.StateStore, configLocks)
		diags = diags.Append(fDiags)
		if fDiags.HasErrors() {
			return nil, true, diags
		}

		// If overrides supplied by -backend-config CLI flag, process them
		var configOverride hcl.Body
		if !extraConfig.Empty() {
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
			configOverride, overrideDiags = c.backendConfigOverrideBody(extraConfig, stateStoreSchema.Body)
			diags = diags.Append(overrideDiags)
			if overrideDiags.HasErrors() {
				return nil, true, diags
			}
		}

		opts = &BackendOpts{
			StateStoreConfig: root.StateStore,
			Locks:            locks,
			ProviderFactory:  factory,
			ConfigOverride:   configOverride,
			Init:             true,
			ViewType:         viewType,
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

		var configOverride hcl.Body
		if len(*extraConfig.Items) > 0 {
			var overrideDiags tfdiags.Diagnostics
			configOverride, overrideDiags = c.backendConfigOverrideBody(extraConfig, backendSchema)
			diags = diags.Append(overrideDiags)
			if overrideDiags.HasErrors() {
				return nil, true, diags
			}
		}

		opts = &BackendOpts{
			BackendConfig:  backendConfig,
			ConfigOverride: configOverride,
			Init:           true,
			ViewType:       viewType,
		}

	default:
		// No config; defaults to local state storage

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

		opts = &BackendOpts{
			Init:     true,
			ViewType: viewType,
		}
	}

	back, backDiags := c.Backend(opts)
	diags = diags.Append(backDiags)
	return back, true, diags
}

// getProviders determines what providers are required given configuration and state data. The method downloads any missing providers
// and replaces the contents of the dependency lock file if any changes happen.
// The calling code is expected to have loaded the complete module tree and read the state file, and passes that data into this method.
//
// This method outputs to the provided view. The returned `output` boolean lets calling code know if anything has been rendered via the view.
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
			// check if required provider dependencies change
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
			view.Output(views.LockInfo)
		} else {
			view.Output(views.DependenciesLockChangesInfo)
		}

		moreDiags = c.replaceLockedDependencies(newLocks)
		diags = diags.Append(moreDiags)
	}

	return true, false, diags
}

// getProvidersFromConfig determines what providers are required by the given configuration data.
// The method downloads any missing providers that aren't already downloaded and then returns
// dependency lock data based on the configuration.
// The dependency lock file itself isn't updated here.
func (c *InitCommand) getProvidersFromConfig(ctx context.Context, config *configs.Config, upgrade bool, pluginDirs []string, flagLockfile string, view views.Init) (output bool, resultingLocks *depsfile.Locks, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "install providers from config")
	defer span.End()

	// Dev overrides cause the result of "terraform init" to be irrelevant for
	// any overridden providers, so we'll warn about it to avoid later
	// confusion when Terraform ends up using a different provider than the
	// lock file called for.
	diags = diags.Append(c.providerDevOverrideInitWarnings())

	// Collect the provider dependencies from the configuration.
	reqs, hclDiags := config.ProviderRequirements()
	diags = diags.Append(hclDiags)
	if hclDiags.HasErrors() {
		return false, nil, diags
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
	if diags.HasErrors() {
		return false, nil, diags
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

	evts := c.prepareInstallerEvents(ctx, reqs, diags, inst, view, views.InitializingProviderPluginFromConfigMessage, views.ReusingPreviousVersionInfo)
	ctx = evts.OnContext(ctx)

	mode := providercache.InstallNewProvidersOnly
	if upgrade {
		if flagLockfile == "readonly" {
			diags = diags.Append(fmt.Errorf("The -upgrade flag conflicts with -lockfile=readonly."))
			view.Diagnostics(diags)
			return true, nil, diags
		}

		mode = providercache.InstallUpgrades
	}

	// Previous locks from dep locks file are needed so we don't re-download any providers
	previousLocks, moreDiags := c.lockedDependencies()
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return false, nil, diags
	}

	// Determine which required providers are already downloaded, and download any
	// new providers or newer versions of providers
	configLocks, err := inst.EnsureProviderVersions(ctx, previousLocks, reqs, mode)
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

	return true, configLocks, diags
}

// getProvidersFromState determines what providers are required by the given state data.
// The method downloads any missing providers that aren't already downloaded and then returns
// dependency lock data based on the state.
// The calling code is assumed to have already called getProvidersFromConfig, which is used to
// supply the configLocks argument.
// The dependency lock file itself isn't updated here.
func (c *InitCommand) getProvidersFromState(ctx context.Context, state *states.State, configLocks *depsfile.Locks, upgrade bool, pluginDirs []string, flagLockfile string, view views.Init) (output bool, resultingLocks *depsfile.Locks, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "install providers from state")
	defer span.End()

	// Dev overrides cause the result of "terraform init" to be irrelevant for
	// any overridden providers, so we'll warn about it to avoid later
	// confusion when Terraform ends up using a different provider than the
	// lock file called for.
	diags = diags.Append(c.providerDevOverrideInitWarnings())

	if state == nil {
		// if there is no state there are no providers to get
		return true, depsfile.NewLocks(), nil
	}
	reqs := state.ProviderRequirements()

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

	// The locks below are used to avoid re-downloading any providers in the
	// second download step.
	// We combine any locks from the dependency lock file and locks identified
	// from the configuration
	var moreDiags tfdiags.Diagnostics
	previousLocks, moreDiags := c.lockedDependencies()
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return false, nil, diags
	}
	inProgressLocks := c.mergeLockedDependencies(configLocks, previousLocks)

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

	// Because we're currently just streaming a series of events sequentially
	// into the terminal, we're showing only a subset of the events to keep
	// things relatively concise. Later it'd be nice to have a progress UI
	// where statuses update in-place, but we can't do that as long as we
	// are shimming our vt100 output to the legacy console API on Windows.
	evts := c.prepareInstallerEvents(ctx, reqs, diags, inst, view, views.InitializingProviderPluginFromStateMessage, views.ReusingVersionIdentifiedFromConfig)
	ctx = evts.OnContext(ctx)

	mode := providercache.InstallNewProvidersOnly

	// We don't handle upgrade flags here, i.e. what happens at this point in getProvidersFromConfig:
	// > We cannot upgrade a provider used only by the state, as there are no version constraints in state.
	//    > Given the overlap between providers in the config and state, using the upgrade mode here
	//      would remove the effects of version constraints from the config.
	// > Any validation of CLI flag usage is already done in getProvidersFromConfig

	newLocks, err := inst.EnsureProviderVersions(ctx, inProgressLocks, reqs, mode)
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
// The calling code is expected to provide the previous locks (if any) and the two sets of locks determined from
// configuration and state data.
func (c *InitCommand) saveDependencyLockFile(previousLocks, configLocks, stateLocks *depsfile.Locks, flagLockfile string, view views.Init) (output bool, diags tfdiags.Diagnostics) {

	// Get the combination of config and state locks
	newLocks := c.mergeLockedDependencies(configLocks, stateLocks)

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
			view.Output(views.LockInfo)
			output = true
		} else {
			view.Output(views.DependenciesLockChangesInfo)
			output = true
		}
		lockFileDiags := c.replaceLockedDependencies(newLocks)
		diags = diags.Append(lockFileDiags)
	}
	return output, diags
}

// prepareInstallerEvents returns an instance of *providercache.InstallerEvents. This struct defines callback functions that will be executed
// when a specific type of event occurs during provider installation.
// The calling code needs to provide a tfdiags.Diagnostics collection, so that provider installation code returns diags to the calling code using closures
func (c *InitCommand) prepareInstallerEvents(ctx context.Context, reqs providerreqs.Requirements, diags tfdiags.Diagnostics, inst *providercache.Installer, view views.Init, initMsg views.InitMessageCode, reuseMsg views.InitMessageCode) *providercache.InstallerEvents {

	// Because we're currently just streaming a series of events sequentially
	// into the terminal, we're showing only a subset of the events to keep
	// things relatively concise. Later it'd be nice to have a progress UI
	// where statuses update in-place, but we can't do that as long as we
	// are shimming our vt100 output to the legacy console API on Windows.
	events := &providercache.InstallerEvents{
		PendingProviders: func(reqs map[addrs.Provider]getproviders.VersionConstraints) {
			view.Output(initMsg)
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
				view.LogInitMessage(reuseMsg, provider.ForDisplay())
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
			c.incompleteProviders = append(c.incompleteProviders, provider.ForDisplay())
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

	return events
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
                          initializing a new state backend. This is
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
