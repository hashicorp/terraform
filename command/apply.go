package command

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	ShutdownCh  <-chan struct{}
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui
}

func (c *ApplyCommand) Run(args []string) int {
	var init bool
	var statePath, stateOutPath string

	cmdFlags := flag.NewFlagSet("apply", flag.ContinueOnError)
	cmdFlags.BoolVar(&init, "init", false, "init")
	cmdFlags.StringVar(&statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&stateOutPath, "state-out", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The apply command expacts at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		var err error
		configPath, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	// If we don't specify an output path, default to out normal state
	// path.
	if stateOutPath == "" {
		stateOutPath = statePath
	}

	// The state path to use to generate a plan. If we're initializing
	// a new infrastructure, then we don't use a state path.
	planStatePath := statePath
	if init {
		planStatePath = ""
	}

	// Build the context based on the arguments given
	c.ContextOpts.Hooks = append(c.ContextOpts.Hooks, &UiHook{Ui: c.Ui})
	ctx, err := ContextArg(configPath, planStatePath, c.ContextOpts)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	// Start the apply in a goroutine so that we can be interrupted.
	var state *terraform.State
	var applyErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		state, applyErr = ctx.Apply()
	}()

	// Wait for the apply to finish or for us to be interrupted so
	// we can handle it properly.
	err = nil
	select {
	case <-c.ShutdownCh:
		c.Ui.Output("Interrupt received. Gracefully shutting down...")

		// Stop execution
		ctx.Stop()

		// Still get the result, since there is still one
		select {
		case <-c.ShutdownCh:
			c.Ui.Error(
				"Two interrupts received. Exiting immediately. Note that data\n" +
					"loss may have occurred.")
			return 1
		case <-doneCh:
		}
	case <-doneCh:
	}

	if state != nil {
		// Write state out to the file
		f, err := os.Create(stateOutPath)
		if err == nil {
			err = terraform.WriteState(state, f)
			f.Close()
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to save state: %s", err))
			return 1
		}
	}

	if applyErr != nil {
		c.Ui.Error(fmt.Sprintf("Error applying plan: %s", applyErr))
		return 1
	}

	c.Ui.Output(strings.TrimSpace(state.String()))

	return 0
}

func (c *ApplyCommand) Help() string {
	helpText := `
Usage: terraform apply [options] [dir]

  Builds or changes infrastructure according to Terraform configuration
  files .

Options:

  -init                  If specified, new infrastructure can be built (no
                         previous state). This is just a safety switch
                         to prevent accidentally spinning up a new
                         infrastructure.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) Synopsis() string {
	return "Builds or changes infrastructure"
}
