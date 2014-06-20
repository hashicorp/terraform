package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	TFConfig *terraform.Config
	Ui       cli.Ui
}

func (c *ApplyCommand) Run(args []string) int {
	var init bool
	var statePath, stateOutPath string

	cmdFlags := flag.NewFlagSet("apply", flag.ContinueOnError)
	cmdFlags.BoolVar(&init, "init", false, "init")
	cmdFlags.StringVar(&statePath, "state", "terraform.tfstate", "path")
	cmdFlags.StringVar(&stateOutPath, "state-out", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error(
			"The apply command expects only one argument with the path\n" +
				"to a Terraform configuration.\n")
		cmdFlags.Usage()
		return 1
	}

	if statePath == "" {
		c.Ui.Error("-state cannot be blank")
		return 1
	}
	if stateOutPath == "" {
		stateOutPath = statePath
	}
	if !init {
		if _, err := os.Stat(statePath); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"There was an error reading the state file. The path\n"+
					"and error are shown below. If you're trying to build a\n"+
					"brand new infrastructure, explicitly pass the '-init'\n"+
					"flag to Terraform to tell it it is okay to build new\n"+
					"state.\n\n"+
					"Path: %s\n"+
					"Error: %s",
				statePath,
				err))
			return 1
		}
	}

	// Load up the state
	var state *terraform.State
	if !init {
		f, err := os.Open(statePath)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
			return 1
		}

		state, err = terraform.ReadState(f)
		f.Close()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error loading state: %s", err))
			return 1
		}
	}

	b, err := config.Load(args[0])
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading blueprint: %s", err))
		return 1
	}

	tfconfig := c.TFConfig
	tfconfig.Config = b

	tf, err := terraform.New(tfconfig)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing Terraform: %s", err))
		return 1
	}

	_, err = tf.Plan(state)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running plan: %s", err))
		return 1
	}

	state, err = tf.Apply(state, nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error applying plan: %s", err))
		return 1
	}

	// Write state out to the file
	f, err := os.Create(stateOutPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to save state: %s", err))
		return 1
	}
	defer f.Close()

	if err := terraform.WriteState(state, f); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to save state: %s", err))
		return 1
	}

	c.Ui.Output(strings.TrimSpace(state.String()))

	return 0
}

func (c *ApplyCommand) Help() string {
	helpText := `
Usage: terraform apply [terraform.tf]

  Builds or changes infrastructure according to the Terraform configuration
  file.

Options:

  -init                     If specified, it is okay to build brand new
                            infrastructure (with no state file specified).

  -state=terraform.tfstate  Path to the state file to build off of. This file
                            will also be written to with updated state unless
                            -state-out is specified.

  -state-out=file.tfstate   Path to save the new state. If not specified, the
                            -state value will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) Synopsis() string {
	return "Builds or changes infrastructure according to Terrafiles"
}
