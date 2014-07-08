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

	planStatePath := statePath
	if init {
		planStatePath = ""
	}

	// Initialize Terraform right away
	c.ContextOpts.Hooks = append(c.ContextOpts.Hooks, &UiHook{Ui: c.Ui})
	ctx, err := ContextArg(configPath, planStatePath, c.ContextOpts)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	var state *terraform.State
	var applyErr error
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		state, applyErr = ctx.Apply()
	}()

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
