package command

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Meta

	ShutdownCh <-chan struct{}
}

func (c *ApplyCommand) Run(args []string) int {
	var init bool
	var statePath, stateOutPath string

	args = c.Meta.process(args)

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

	// Prepare the extra hooks to count resources
	countHook := new(CountHook)
	c.Meta.extraHooks = []terraform.Hook{countHook}

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
	ctx, err := c.Context(configPath, planStatePath, true)
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
		c.Ui.Error(fmt.Sprintf(
			"Error applying plan:\n\n"+
				"%s\n\n"+
				"Terraform does not automatically rollback in the face of errors.\n"+
				"Instead, your Terraform state file has been partially updated with\n"+
				"any resources that successfully completed. Please address the error\n"+
				"above and apply again to incrementally change your infrastructure.",
			applyErr))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		"[reset][bold][green]\n"+
			"Apply complete! Resources: %d added, %d changed, %d destroyed.",
		countHook.Added,
		countHook.Changed,
		countHook.Removed)))

	if countHook.Added > 0 || countHook.Changed > 0 {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
			"[reset]\n"+
				"The state of your infrastructure has been saved to the path\n"+
				"below. This state is required to modify and destroy your\n"+
				"infrastructure, so keep it safe. To inspect the complete state\n"+
				"use the `terraform show` command.\n\n"+
				"State path: %s",
			stateOutPath)))
	}

	// If we have outputs, then output those at the end.
	if len(state.Outputs) > 0 {
		outputBuf := new(bytes.Buffer)
		outputBuf.WriteString("[reset][bold][green]\nOutputs:\n\n")

		// Output the outputs in alphabetical order
		keyLen := 0
		keys := make([]string, 0, len(state.Outputs))
		for key, _ := range state.Outputs {
			keys = append(keys, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := state.Outputs[k]

			outputBuf.WriteString(fmt.Sprintf(
				"  %s%s = %s\n",
				k,
				strings.Repeat(" ", keyLen-len(k)),
				v))
		}

		c.Ui.Output(c.Colorize().Color(
			strings.TrimSpace(outputBuf.String())))
	}

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

  -no-color              If specified, output won't contain any color.

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
