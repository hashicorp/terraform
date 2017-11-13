package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/hashicorp/terraform/command"
	pluginDiscovery "github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/mitchellh/cli"
)

// runningInAutomationEnvName gives the name of an environment variable that
// can be set to any non-empty value in order to suppress certain messages
// that assume that Terraform is being run from a command prompt.
const runningInAutomationEnvName = "TF_IN_AUTOMATION"

// Commands is the mapping of all the available Terraform commands.
var Commands map[string]cli.CommandFactory
var PlumbingCommands map[string]struct{}

// Ui is the cli.Ui used for communicating to the outside world.
var Ui cli.Ui

const (
	ErrorPrefix  = "e:"
	OutputPrefix = "o:"
)

func initCommands(config *Config) {
	var inAutomation bool
	if v := os.Getenv(runningInAutomationEnvName); v != "" {
		inAutomation = true
	}

	credsSrc := credentialsSource(config)
	services := disco.NewDisco()
	services.SetCredentialsSource(credsSrc)
	for userHost, hostConfig := range config.Hosts {
		host, err := svchost.ForComparison(userHost)
		if err != nil {
			// We expect the config was already validated by the time we get
			// here, so we'll just ignore invalid hostnames.
			continue
		}
		services.ForceHostServices(host, hostConfig.Services)
	}

	dataDir := os.Getenv("TF_DATA_DIR")

	meta := command.Meta{
		Color:            true,
		GlobalPluginDirs: globalPluginDirs(),
		PluginOverrides:  &PluginOverrides,
		Ui:               Ui,

		Services:    services,
		Credentials: credsSrc,

		RunningInAutomation: inAutomation,
		PluginCacheDir:      config.PluginCacheDir,
		OverrideDataDir:     dataDir,
	}

	// The command list is included in the terraform -help
	// output, which is in turn included in the docs at
	// website/source/docs/commands/index.html.markdown; if you
	// add, remove or reclassify commands then consider updating
	// that to match.

	PlumbingCommands = map[string]struct{}{
		"state":        struct{}{}, // includes all subcommands
		"debug":        struct{}{}, // includes all subcommands
		"force-unlock": struct{}{},
	}

	Commands = map[string]cli.CommandFactory{
		"apply": func() (cli.Command, error) {
			return &command.ApplyCommand{
				Meta:       meta,
				ShutdownCh: makeShutdownCh(),
			}, nil
		},

		"console": func() (cli.Command, error) {
			return &command.ConsoleCommand{
				Meta:       meta,
				ShutdownCh: makeShutdownCh(),
			}, nil
		},

		"destroy": func() (cli.Command, error) {
			return &command.ApplyCommand{
				Meta:       meta,
				Destroy:    true,
				ShutdownCh: makeShutdownCh(),
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

		"internal-plugin": func() (cli.Command, error) {
			return &command.InternalPluginCommand{
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

		"validate": func() (cli.Command, error) {
			return &command.ValidateCommand{
				Meta: meta,
			}, nil
		},

		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Meta:              meta,
				Revision:          GitCommit,
				Version:           Version,
				VersionPrerelease: VersionPrerelease,
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

		"debug": func() (cli.Command, error) {
			return &command.DebugCommand{
				Meta: meta,
			}, nil
		},

		"debug json2dot": func() (cli.Command, error) {
			return &command.DebugJSON2DotCommand{
				Meta: meta,
			}, nil
		},

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

func credentialsSource(config *Config) auth.CredentialsSource {
	creds := auth.NoCredentials
	if len(config.Credentials) > 0 {
		staticTable := map[svchost.Hostname]map[string]interface{}{}
		for userHost, creds := range config.Credentials {
			host, err := svchost.ForComparison(userHost)
			if err != nil {
				// We expect the config was already validated by the time we get
				// here, so we'll just ignore invalid hostnames.
				continue
			}
			staticTable[host] = creds
		}
		creds = auth.StaticCredentialsSource(staticTable)
	}

	for helperType, helperConfig := range config.CredentialsHelpers {
		log.Printf("[DEBUG] Searching for credentials helper named %q", helperType)
		available := pluginDiscovery.FindPlugins("credentials", globalPluginDirs())
		available = available.WithName(helperType)
		if available.Count() == 0 {
			log.Printf("[ERROR] Unable to find credentials helper %q; ignoring", helperType)
			break
		}

		selected := available.Newest()

		helperSource := auth.HelperProgramCredentialsSource(selected.Path, helperConfig.Args...)
		creds = auth.Credentials{
			creds,
			auth.CachingCredentialsSource(helperSource), // cached because external operation may be slow/expensive
		}

		// There should only be zero or one "credentials_helper" blocks. We
		// assume that the config was validated earlier and so we don't check
		// for extras here.
		break
	}

	return creds
}
