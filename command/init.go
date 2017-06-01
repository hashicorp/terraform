package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/variables"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta
}

func (c *InitCommand) Run(args []string) int {
	var flagBackend, flagGet bool
	var flagConfigExtra map[string]interface{}

	args = c.Meta.process(args, false)
	cmdFlags := c.flagSet("init")
	cmdFlags.BoolVar(&flagBackend, "backend", true, "")
	cmdFlags.Var((*variables.FlagAny)(&flagConfigExtra), "backend-config", "")
	cmdFlags.BoolVar(&flagGet, "get", true, "")
	cmdFlags.BoolVar(&c.forceInitCopy, "force-copy", false, "suppress prompts about copying state data")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&c.reconfigure, "reconfigure", false, "reconfigure")

	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Validate the arg count
	args = cmdFlags.Args()
	if len(args) > 2 {
		c.Ui.Error("The init command expects at most two arguments.\n")
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
	var path string
	var source string
	switch len(args) {
	case 0:
		path = pwd
	case 1:
		path = pwd
		source = args[0]
	case 2:
		source = args[0]
		path = args[1]
	default:
		panic("assertion failed on arg count")
	}

	// Set the state out path to be the path requested for the module
	// to be copied. This ensures any remote states gets setup in the
	// proper directory.
	c.Meta.dataDir = filepath.Join(path, DefaultDataDir)

	// This will track whether we outputted anything so that we know whether
	// to output a newline before the success message
	var header bool

	// If we have a source, copy it
	if source != "" {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset][bold]"+
				"Initializing configuration from: %q...", source)))
		if err := c.copySource(path, source, pwd); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error copying source: %s", err))
			return 1
		}

		header = true
	}

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

	// If we're performing a get or loading the backend, then we perform
	// some extra tasks.
	if flagGet || flagBackend {
		// Load the configuration in this directory so that we can know
		// if we have anything to get or any backend to configure. We do
		// this to improve the UX. Practically, we could call the functions
		// below without checking this to the same effect.
		conf, err := config.LoadDir(path)
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error loading configuration: %s", err))
			return 1
		}

		// If we requested downloading modules and have modules in the config
		if flagGet && len(conf.Modules) > 0 {
			header = true

			c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
				"[reset][bold]" +
					"Downloading modules (if any)...")))
			if err := getModules(&c.Meta, path, module.GetModeGet); err != nil {
				c.Ui.Error(fmt.Sprintf(
					"Error downloading modules: %s", err))
				return 1
			}
		}

		// If we're requesting backend configuration and configure it
		if flagBackend {
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
				ConfigPath:  path,
				ConfigExtra: flagConfigExtra,
				Init:        true,
			}
			if _, err := c.Backend(opts); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
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

func (c *InitCommand) copySource(dst, src, pwd string) error {
	// Verify the directory is empty
	if empty, err := config.IsEmptyDir(dst); err != nil {
		return fmt.Errorf("Error checking on destination path: %s", err)
	} else if !empty {
		return fmt.Errorf(strings.TrimSpace(errInitCopyNotEmpty))
	}

	// Detect
	source, err := getter.Detect(src, pwd, getter.Detectors)
	if err != nil {
		return fmt.Errorf("Error with module source: %s", err)
	}

	// Get it!
	return module.GetCopy(dst, source)
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: terraform init [options] [SOURCE] [PATH]

  Initialize a new or existing Terraform environment by creating
  initial files, loading any remote state, downloading modules, etc.

  This is the first command that should be run for any new or existing
  Terraform configuration per machine. This sets up all the local data
  necessary to run Terraform that is typically not committed to version
  control.

  This command is always safe to run multiple times. Though subsequent runs
  may give errors, this command will never blow away your environment or state.
  Even so, if you have important information, please back it up prior to
  running this command just in case.

  If no arguments are given, the configuration in this working directory
  is initialized.

  If one or two arguments are given, the first is a SOURCE of a module to
  download to the second argument PATH. After downloading the module to PATH,
  the configuration will be initialized as if this command were called pointing
  only to that PATH. PATH must be empty of any Terraform files. Any
  conflicting non-Terraform files will be overwritten. The module download
  is a copy. If you're downloading a module from Git, it will not preserve
  Git history.

Options:

  -backend=true        Configure the backend for this environment.

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

  -input=true          Ask for input if necessary. If false, will error if
                       input was required.

  -lock=true           Lock the state file when locking is supported.

  -lock-timeout=0s     Duration to retry a state lock.

  -no-color            If specified, output won't contain any color.

  -reconfigure          Reconfigure the backend, ignoring any saved configuration.
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
rerun this command to reinitialize your environment. If you forget, other
commands will detect it and remind you to do so if necessary.
`
