package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/variables"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta

	// providerInstaller is used to download and install providers that
	// aren't found locally. This uses a discovery.ProviderInstaller instance
	// by default, but it can be overridden here as a way to mock fetching
	// providers for tests.
	providerInstaller discovery.Installer
}

func (c *InitCommand) Run(args []string) int {
	var flagBackend, flagGet, flagGetPlugins, flagUpgrade bool
	var flagConfigExtra map[string]interface{}

	args = c.Meta.process(args, false)
	cmdFlags := c.flagSet("init")
	cmdFlags.BoolVar(&flagBackend, "backend", true, "")
	cmdFlags.Var((*variables.FlagAny)(&flagConfigExtra), "backend-config", "")
	cmdFlags.BoolVar(&flagGet, "get", true, "")
	cmdFlags.BoolVar(&flagGetPlugins, "get-plugins", true, "")
	cmdFlags.BoolVar(&c.forceInitCopy, "force-copy", false, "suppress prompts about copying state data")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&c.reconfigure, "reconfigure", false, "reconfigure")
	cmdFlags.BoolVar(&flagUpgrade, "upgrade", false, "")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// set getProvider if we don't have a test version already
	if c.providerInstaller == nil {
		c.providerInstaller = &discovery.ProviderInstaller{
			Dir: c.pluginDir(),

			PluginProtocolVersion: plugin.Handshake.ProtocolVersion,
		}
	}

	// Validate the arg count
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The init command expects at most one argument.\n")
		cmdFlags.Usage()
		return 1
	}

	// Get our pwd. We don't always need it but always getting it is easier
	// than the logic to determine if it is or isn't needed.
	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		return 1
	}

	// Get the path and source module to copy
	path := pwd
	if len(args) == 1 {
		path = args[0]
	}
	// Set the state out path to be the path requested for the module
	// to be copied. This ensures any remote states gets setup in the
	// proper directory.
	c.Meta.dataDir = filepath.Join(path, DefaultDataDir)

	// This will track whether we outputted anything so that we know whether
	// to output a newline before the success message
	var header bool

	// If our directory is empty, then we're done. We can't get or setup
	// the backend with an empty directory.
	if empty, err := config.IsEmptyDir(path); err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error checking configuration: %s", err))
		return 1
	} else if empty {
		c.Ui.Output(c.Colorize().Color(strings.TrimSpace(outputInitEmpty)))
		return 0
	}

	var back backend.Backend

	// If we're performing a get or loading the backend, then we perform
	// some extra tasks.
	if flagGet || flagBackend {
		conf, err := c.Config(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error loading configuration: %s", err))
			return 1
		}

		// If we requested downloading modules and have modules in the config
		if flagGet && len(conf.Modules) > 0 {
			header = true

			getMode := module.GetModeGet
			if flagUpgrade {
				getMode = module.GetModeUpdate
				c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
					"[reset][bold]Upgrading modules...")))
			} else {
				c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
					"[reset][bold]Downloading modules...")))
			}

			if err := getModules(&c.Meta, path, getMode); err != nil {
				c.Ui.Error(fmt.Sprintf(
					"Error downloading modules: %s", err))
				return 1
			}

		}

		// If we're requesting backend configuration or looking for required
		// plugins, load the backend
		if flagBackend || flagGetPlugins {
			header = true

			// Only output that we're initializing a backend if we have
			// something in the config. We can be UNSETTING a backend as well
			// in which case we choose not to show this.
			if conf.Terraform != nil && conf.Terraform.Backend != nil {
				c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
					"[reset][bold]" +
						"Initializing the backend...")))
			}

			opts := &BackendOpts{
				Config:      conf,
				ConfigExtra: flagConfigExtra,
				Init:        true,
			}
			if back, err = c.Backend(opts); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
		}
	}

	// Now that we have loaded all modules, check the module tree for missing providers
	if flagGetPlugins {
		sMgr, err := back.State(c.Workspace())
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error loading state: %s", err))
			return 1
		}

		if err := sMgr.RefreshState(); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error refreshing state: %s", err))
			return 1
		}

		c.Ui.Output(c.Colorize().Color(
			"[reset][bold]Initializing provider plugins...",
		))

		err = c.getProviders(path, sMgr.State(), flagUpgrade)
		if err != nil {
			// this function provides its own output
			log.Printf("[ERROR] %s", err)
			return 1
		}
	}

	// If we outputted information, then we need to output a newline
	// so that our success message is nicely spaced out from prior text.
	if header {
		c.Ui.Output("")
	}

	c.Ui.Output(c.Colorize().Color(strings.TrimSpace(outputInitSuccess)))

	return 0
}

// Load the complete module tree, and fetch any missing providers.
// This method outputs its own Ui.
func (c *InitCommand) getProviders(path string, state *terraform.State, upgrade bool) error {
	mod, err := c.Module(path)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting plugins: %s", err))
		return err
	}

	if err := mod.Validate(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting plugins: %s", err))
		return err
	}

	available := c.providerPluginSet()
	requirements := terraform.ModuleTreeDependencies(mod, state).AllPluginRequirements()
	missing := c.missingPlugins(available, requirements)

	var errs error
	for provider, reqd := range missing {
		c.Ui.Output(fmt.Sprintf("- downloading plugin for provider %q...", provider))
		_, err := c.providerInstaller.Get(provider, reqd.Versions)

		if err != nil {
			c.Ui.Error(fmt.Sprintf(errProviderNotFound, err, provider, reqd.Versions))
			errs = multierror.Append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}

	// With all the providers downloaded, we'll generate our lock file
	// that ensures the provider binaries remain unchanged until we init
	// again. If anything changes, other commands that use providers will
	// fail with an error instructing the user to re-run this command.
	available = c.providerPluginSet() // re-discover to see newly-installed plugins
	chosen := choosePlugins(available, requirements)
	digests := map[string][]byte{}
	for name, meta := range chosen {
		digest, err := meta.SHA256()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("failed to read provider plugin %s: %s", meta.Path, err))
			return err
		}
		digests[name] = digest
	}
	err = c.providerPluginsLock().Write(digests)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("failed to save provider manifest: %s", err))
		return err
	}

	// If any providers have "floating" versions (completely unconstrained)
	// we'll suggest the user constrain with a pessimistic constraint to
	// avoid implicitly adopting a later major release.
	constraintSuggestions := make(map[string]discovery.ConstraintStr)
	for name, meta := range chosen {
		req := requirements[name]
		if req == nil {
			// should never happen, but we don't want to crash here, so we'll
			// be cautious.
			continue
		}

		if req.Versions.Unconstrained() {
			// meta.Version.MustParse is safe here because our "chosen" metas
			// were already filtered for validity of versions.
			constraintSuggestions[name] = meta.Version.MustParse().MinorUpgradeConstraintStr()
		}
	}
	if len(constraintSuggestions) != 0 {
		names := make([]string, 0, len(constraintSuggestions))
		for name := range constraintSuggestions {
			names = append(names, name)
		}
		sort.Strings(names)

		c.Ui.Output(outputInitProvidersUnconstrained)
		for _, name := range names {
			c.Ui.Output(fmt.Sprintf("* provider.%s: version = %q", name, constraintSuggestions[name]))
		}
	}

	return nil
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: terraform init [options] [DIR]

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

  If no arguments are given, the configuration in this working directory
  is initialized.

Options:

  -backend=true        Configure the backend for this configuration.

  -backend-config=path This can be either a path to an HCL file with key/value
                       assignments (same format as terraform.tfvars) or a
                       'key=value' format. This is merged with what is in the
                       configuration file. This can be specified multiple
                       times. The backend type must be in the configuration
                       itself.

  -force-copy          Suppress prompts about copying state data. This is
                       equivalent to providing a "yes" to all confirmation
                       prompts.

  -get=true            Download any modules for this configuration.

  -get-plugins=true    Download any missing plugins for this configuration.

  -input=true          Ask for input if necessary. If false, will error if
                       input was required.

  -lock=true           Lock the state file when locking is supported.

  -lock-timeout=0s     Duration to retry a state lock.

  -no-color            If specified, output won't contain any color.

  -reconfigure         Reconfigure the backend, ignoring any saved configuration.

  -upgrade=false       If installing modules (-get) or plugins (-get-plugins),
                       ignore previously-downloaded objects and install the
                       latest version allowed within configured constraints.
`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initialize a new or existing Terraform configuration"
}

const errInitCopyNotEmpty = `
The destination path contains Terraform configuration files. The init command
with a SOURCE parameter can only be used on a directory without existing
Terraform files.

Please resolve this issue and try again.
`

const outputInitEmpty = `
[reset][bold]Terraform initialized in an empty directory![reset]

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitSuccess = `
[reset][bold][green]Terraform has been successfully initialized![reset][green]

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitProvidersUnconstrained = `
The following providers do not have any version constraints in configuration,
so the latest version was installed.

To prevent automatic upgrades to new major versions that may contain breaking
changes, it is recommended to add version = "..." constraints to the
corresponding provider blocks in configuration, with the constraint strings
suggested below.
`

const errProviderNotFound = `
[reset][red]%[1]s

[reset][bold][red]Error: Satisfying %[2]q, provider not found

[reset][red]A version of the %[2]q provider that satisfies all version
constraints could not be found. The requested version
constraints are shown below.

%[2]s = %[3]q[reset]
`
