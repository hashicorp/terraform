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
	var stateOutPath string

	cmdFlags := flag.NewFlagSet("apply", flag.ContinueOnError)
	cmdFlags.BoolVar(&init, "init", false, "init")
	cmdFlags.StringVar(&stateOutPath, "out", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 2 {
		c.Ui.Error("The apply command expects two arguments.\n")
		cmdFlags.Usage()
		return 1
	}

	statePath := args[0]
	configPath := args[1]

	if stateOutPath == "" {
		stateOutPath = statePath
	}

	// Initialize Terraform right away
	c.TFConfig.Hooks = append(c.TFConfig.Hooks, &UiHook{Ui: c.Ui})
	tf, err := terraform.New(c.TFConfig)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error initializing Terraform: %s", err))
		return 1
	}

	// Attempt to read a plan from the path given. This is how we test that
	// it is a plan or not (kind of jank, but if it quacks like a duck...)
	var plan *terraform.Plan
	f, err := os.Open(configPath)
	if err == nil {
		plan, err = terraform.ReadPlan(f)
		f.Close()
		if err != nil {
			// Make sure the plan is nil so that we try to load as
			// configuration.
			plan = nil
		}
	}

	if plan == nil {
		// No plan was given, so we're loading from configuration. Generate
		// the plan given the configuration.
		plan, err = c.configToPlan(tf, init, statePath, configPath)
		if err != nil {
			c.Ui.Error(err.Error())
			return 1
		}
	}

	state, err := tf.Apply(plan)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error applying plan: %s", err))
		return 1
	}

	// Write state out to the file
	f, err = os.Create(stateOutPath)
	if err == nil {
		err = terraform.WriteState(state, f)
		f.Close()
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to save state: %s", err))
		return 1
	}

	c.Ui.Output(strings.TrimSpace(state.String()))

	return 0
}

func (c *ApplyCommand) Help() string {
	helpText := `
Usage: terraform apply [options] STATE PATH

  Builds or changes infrastructure according to the Terraform configuration
  file.

Options:

  -init                     If specified, it is okay to build brand new
                            infrastructure (with no state file specified).

  -out=file.tfstate         Path to save the new state. If not specified, the
                            state path argument will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) Synopsis() string {
	return "Builds or changes infrastructure"
}

func (c *ApplyCommand) configToPlan(
	tf *terraform.Terraform,
	init bool,
	statePath string,
	configPath string) (*terraform.Plan, error) {
	if !init {
		if _, err := os.Stat(statePath); err != nil {
			return nil, fmt.Errorf(
				"There was an error reading the state file. The path\n"+
					"and error are shown below. If you're trying to build a\n"+
					"brand new infrastructure, explicitly pass the '-init'\n"+
					"flag to Terraform to tell it it is okay to build new\n"+
					"state.\n\n"+
					"Path: %s\n"+
					"Error: %s",
				statePath,
				err)
		}
	}

	// Load up the state
	var state *terraform.State
	if !init {
		f, err := os.Open(statePath)
		if err == nil {
			state, err = terraform.ReadState(f)
			f.Close()
		}

		if err != nil {
			return nil, fmt.Errorf("Error loading state: %s", err)
		}
	}

	config, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("Error loading config: %s", err)
	}

	plan, err := tf.Plan(config, state, nil)
	if err != nil {
		return nil, fmt.Errorf("Error running plan: %s", err)
	}

	return plan, nil
}
