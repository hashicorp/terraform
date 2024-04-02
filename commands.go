// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/webbrowser"
	"github.com/hashicorp/terraform/internal/getproviders"
	pluginDiscovery "github.com/hashicorp/terraform/internal/plugin/discovery"
	"github.com/hashicorp/terraform/internal/rpcapi"
	"github.com/hashicorp/terraform/internal/terminal"
)

// runningInAutomationEnvName gives the name of an environment variable that
// can be set to any non-empty value in order to suppress certain messages
// that assume that Terraform is being run from a command prompt.
const runningInAutomationEnvName = "TF_IN_AUTOMATION"

// Commands is the mapping of all the available Terraform commands.
var Commands map[string]cli.CommandFactory

// PrimaryCommands is an ordered sequence of the top-level commands (not
// subcommands) that we emphasize at the top of our help output. This is
// ordered so that we can show them in the typical workflow order, rather
// than in alphabetical order. Anything not in this sequence or in the
// HiddenCommands set appears under "all other commands".
var PrimaryCommands []string

// HiddenCommands is a set of top-level commands (not subcommands) that are
// not advertised in the top-level help at all. This is typically because
// they are either just stubs that return an error message about something
// no longer being supported or backward-compatibility aliases for other
// commands.
//
// No commands in the PrimaryCommands sequence should also appear in the
// HiddenCommands set, because that would be rather silly.
var HiddenCommands map[string]struct{}

// Ui is the cli.Ui used for communicating to the outside world.
var Ui cli.Ui

func initCommands(
	ctx context.Context,
	originalWorkingDir string,
	streams *terminal.Streams,
	config *cliconfig.Config,
	services *disco.Disco,
	providerSrc getproviders.Source,
	providerDevOverrides map[addrs.Provider]getproviders.PackageLocalDir,
	unmanagedProviders map[addrs.Provider]*plugin.ReattachConfig,
) {
	var inAutomation bool
	if v := os.Getenv(runningInAutomationEnvName); v != "" {
		inAutomation = true
	}

	for userHost, hostConfig := range config.Hosts {
		host, err := svchost.ForComparison(userHost)
		if err != nil {
			// We expect the config was already validated by the time we get
			// here, so we'll just ignore invalid hostnames.
			continue
		}
		services.ForceHostServices(host, hostConfig.Services)
	}

	configDir, err := cliconfig.ConfigDir()
	if err != nil {
		configDir = "" // No config dir available (e.g. looking up a home directory failed)
	}

	wd := WorkingDir(originalWorkingDir, os.Getenv("TF_DATA_DIR"))

	var browserLauncher webbrowser.Launcher
	if _, ok := os.LookupEnv("TF_BROWSER_ENV"); ok {
		browserLauncher = webbrowser.NewBrowserEnvLauncher()
	} else {
		browserLauncher = webbrowser.NewNativeLauncher()
	}

	meta := command.Meta{
		WorkingDir: wd,
		Streams:    streams,
		View:       views.NewView(streams).SetRunningInAutomation(inAutomation),

		Color:            true,
		GlobalPluginDirs: globalPluginDirs(),
		Ui:               Ui,

		Services:        services,
		BrowserLauncher: browserLauncher,

		RunningInAutomation: inAutomation,
		CLIConfigDir:        configDir,
		PluginCacheDir:      config.PluginCacheDir,

		PluginCacheMayBreakDependencyLockFile: config.PluginCacheMayBreakDependencyLockFile,

		ShutdownCh:    makeShutdownCh(),
		CallerContext: ctx,

		ProviderSource:       providerSrc,
		ProviderDevOverrides: providerDevOverrides,
		UnmanagedProviders:   unmanagedProviders,

		AllowExperimentalFeatures: ExperimentsAllowed(),
	}

	// The command list is included in the terraform -help
	// output, which is in turn included in the docs at
	// website/docs/cli/commands/index.html.markdown; if you
	// add, remove or reclassify commands then consider updating
	// that to match.

	Commands = map[string]cli.CommandFactory{
		"apply": func() (cli.Command, error) {
			return &command.ApplyCommand{
				Meta: meta,
			}, nil
		},

		"console": func() (cli.Command, error) {
			return &command.ConsoleCommand{
				Meta: meta,
			}, nil
		},

		"destroy": func() (cli.Command, error) {
			return &command.ApplyCommand{
				Meta:    meta,
				Destroy: true,
			}, nil
		},

		"env": func() (cli.Command, error) {
			return &command.WorkspaceCommand{
				Meta:       meta,
				LegacyName: true,
			}, nil
		},

		"env list": func() (cli.Command, error) {
			return &command.WorkspaceListCommand{
				Meta:       meta,
				LegacyName: true,
			}, nil
		},

		"env select": func() (cli.Command, error) {
			return &command.WorkspaceSelectCommand{
				Meta:       meta,
				LegacyName: true,
			}, nil
		},

		"env new": func() (cli.Command, error) {
			return &command.WorkspaceNewCommand{
				Meta:       meta,
				LegacyName: true,
			}, nil
		},

		"env delete": func() (cli.Command, error) {
			return &command.WorkspaceDeleteCommand{
				Meta:       meta,
				LegacyName: true,
			}, nil
		},

		"fmt": func() (cli.Command, error) {
			return &command.FmtCommand{
				Meta: meta,
			}, nil
		},

		"get": func() (cli.Command, error) {
			return &command.GetCommand{
				Meta: meta,
			}, nil
		},

		"graph": func() (cli.Command, error) {
			return &command.GraphCommand{
				Meta: meta,
			}, nil
		},

		"import": func() (cli.Command, error) {
			return &command.ImportCommand{
				Meta: meta,
			}, nil
		},

		"init": func() (cli.Command, error) {
			return &command.InitCommand{
				Meta: meta,
			}, nil
		},

		"login": func() (cli.Command, error) {
			return &command.LoginCommand{
				Meta: meta,
			}, nil
		},

		"logout": func() (cli.Command, error) {
			return &command.LogoutCommand{
				Meta: meta,
			}, nil
		},

		"metadata": func() (cli.Command, error) {
			return &command.MetadataCommand{
				Meta: meta,
			}, nil
		},

		"metadata functions": func() (cli.Command, error) {
			return &command.MetadataFunctionsCommand{
				Meta: meta,
			}, nil
		},

		"output": func() (cli.Command, error) {
			return &command.OutputCommand{
				Meta: meta,
			}, nil
		},

		"plan": func() (cli.Command, error) {
			return &command.PlanCommand{
				Meta: meta,
			}, nil
		},

		"providers": func() (cli.Command, error) {
			return &command.ProvidersCommand{
				Meta: meta,
			}, nil
		},

		"providers lock": func() (cli.Command, error) {
			return &command.ProvidersLockCommand{
				Meta: meta,
			}, nil
		},

		"providers mirror": func() (cli.Command, error) {
			return &command.ProvidersMirrorCommand{
				Meta: meta,
			}, nil
		},

		"providers schema": func() (cli.Command, error) {
			return &command.ProvidersSchemaCommand{
				Meta: meta,
			}, nil
		},

		"push": func() (cli.Command, error) {
			return &command.PushCommand{
				Meta: meta,
			}, nil
		},

		"refresh": func() (cli.Command, error) {
			return &command.RefreshCommand{
				Meta: meta,
			}, nil
		},

		"show": func() (cli.Command, error) {
			return &command.ShowCommand{
				Meta: meta,
			}, nil
		},

		"taint": func() (cli.Command, error) {
			return &command.TaintCommand{
				Meta: meta,
			}, nil
		},

		"test": func() (cli.Command, error) {
			return &command.TestCommand{
				Meta: meta,
			}, nil
		},

		"validate": func() (cli.Command, error) {
			return &command.ValidateCommand{
				Meta: meta,
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Meta:              meta,
				Version:           Version,
				VersionPrerelease: VersionPrerelease,
				Platform:          getproviders.CurrentPlatform,
				CheckFunc:         commandVersionCheck,
			}, nil
		},

		"untaint": func() (cli.Command, error) {
			return &command.UntaintCommand{
				Meta: meta,
			}, nil
		},

		"workspace": func() (cli.Command, error) {
			return &command.WorkspaceCommand{
				Meta: meta,
			}, nil
		},

		"workspace list": func() (cli.Command, error) {
			return &command.WorkspaceListCommand{
				Meta: meta,
			}, nil
		},

		"workspace select": func() (cli.Command, error) {
			return &command.WorkspaceSelectCommand{
				Meta: meta,
			}, nil
		},

		"workspace show": func() (cli.Command, error) {
			return &command.WorkspaceShowCommand{
				Meta: meta,
			}, nil
		},

		"workspace new": func() (cli.Command, error) {
			return &command.WorkspaceNewCommand{
				Meta: meta,
			}, nil
		},

		"workspace delete": func() (cli.Command, error) {
			return &command.WorkspaceDeleteCommand{
				Meta: meta,
			}, nil
		},

		//-----------------------------------------------------------
		// Plumbing
		//-----------------------------------------------------------

		"force-unlock": func() (cli.Command, error) {
			return &command.UnlockCommand{
				Meta: meta,
			}, nil
		},

		"state": func() (cli.Command, error) {
			return &command.StateCommand{}, nil
		},

		"state list": func() (cli.Command, error) {
			return &command.StateListCommand{
				Meta: meta,
			}, nil
		},

		"state rm": func() (cli.Command, error) {
			return &command.StateRmCommand{
				StateMeta: command.StateMeta{
					Meta: meta,
				},
			}, nil
		},

		"state mv": func() (cli.Command, error) {
			return &command.StateMvCommand{
				StateMeta: command.StateMeta{
					Meta: meta,
				},
			}, nil
		},

		"state pull": func() (cli.Command, error) {
			return &command.StatePullCommand{
				Meta: meta,
			}, nil
		},

		"state push": func() (cli.Command, error) {
			return &command.StatePushCommand{
				Meta: meta,
			}, nil
		},

		"state show": func() (cli.Command, error) {
			return &command.StateShowCommand{
				Meta: meta,
			}, nil
		},

		"state replace-provider": func() (cli.Command, error) {
			return &command.StateReplaceProviderCommand{
				StateMeta: command.StateMeta{
					Meta: meta,
				},
			}, nil
		},
	}

	if meta.AllowExperimentalFeatures {
		Commands["cloud"] = func() (cli.Command, error) {
			return &command.CloudCommand{
				Meta: meta,
			}, nil
		}

		// "rpcapi" is handled a bit differently because the whole point of
		// this interface is to bypass the CLI layer so wrapping automation can
		// get as-direct-as-possible access to Terraform Core functionality,
		// without interference from behaviors that are intended for CLI
		// end-user convenience. We bypass the "command" package entirely
		// for this command in particular.
		Commands["rpcapi"] = rpcapi.CLICommandFactory(rpcapi.CommandFactoryOpts{
			ExperimentsAllowed: meta.AllowExperimentalFeatures,
			ShutdownCh:         meta.ShutdownCh,
		})
	}

	PrimaryCommands = []string{
		"init",
		"validate",
		"plan",
		"apply",
		"destroy",
	}

	HiddenCommands = map[string]struct{}{
		"env":             {},
		"internal-plugin": {},
		"push":            {},
		"rpcapi":          {},
	}
}

// makeShutdownCh creates an interrupt listener and returns a channel.
// A message will be sent on the channel for every interrupt received.
func makeShutdownCh() <-chan struct{} {
	resultCh := make(chan struct{})

	signalCh := make(chan os.Signal, 4)
	signal.Notify(signalCh, ignoreSignals...)
	signal.Notify(signalCh, forwardSignals...)
	go func() {
		for {
			<-signalCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}

func credentialsSource(config *cliconfig.Config) (auth.CredentialsSource, error) {
	helperPlugins := pluginDiscovery.FindPlugins("credentials", globalPluginDirs())
	return config.CredentialsSource(helperPlugins)
}
