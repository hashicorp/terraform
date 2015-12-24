package command

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/terraform"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Meta

	// If true, then this apply command will become the "destroy"
	// command. It is just like apply but only processes a destroy.
	Destroy bool

	// When this channel is closed, the apply will be cancelled.
	ShutdownCh <-chan struct{}
}

func (c *ApplyCommand) Run(args []string) int {
	var destroyForce, refresh bool
	args = c.Meta.process(args, true)

	cmdName := "apply"
	if c.Destroy {
		cmdName = "destroy"
	}

	cmdFlags := c.Meta.flagSet(cmdName)
	if c.Destroy {
		cmdFlags.BoolVar(&destroyForce, "force", false, "force")
	}
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	cmdFlags.IntVar(
		&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		return 1
	}

	var configPath string
	maybeInit := true
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The apply command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		configPath = pwd
		maybeInit = false
	}

	// Prepare the extra hooks to count resources
	countHook := new(CountHook)
	stateHook := new(StateHook)
	c.Meta.extraHooks = []terraform.Hook{countHook, stateHook}

	if !c.Destroy && maybeInit {
		// Do a detect to determine if we need to do an init + apply.
		if detected, err := getter.Detect(configPath, pwd, getter.Detectors); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Invalid path: %s", err))
			return 1
		} else if !strings.HasPrefix(detected, "file") {
			// If this isn't a file URL then we're doing an init +
			// apply.
			var init InitCommand
			init.Meta = c.Meta
			if code := init.Run([]string{detected}); code != 0 {
				return code
			}

			// Change the config path to be the cwd
			configPath = pwd
		}
	}

	// Build the context based on the arguments given
	ctx, planned, err := c.Context(contextOpts{
		Destroy:     c.Destroy,
		Path:        configPath,
		StatePath:   c.Meta.statePath,
		Parallelism: c.Meta.parallelism,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if c.Destroy && planned {
		c.Ui.Error(fmt.Sprintf(
			"Destroy can't be called with a plan file."))
		return 1
	}
	if !destroyForce && c.Destroy {
		// Default destroy message
		desc := "Terraform will delete all your managed infrastructure.\n" +
			"There is no undo. Only 'yes' will be accepted to confirm."

		// If targets are specified, list those to user
		if c.Meta.targets != nil {
			var descBuffer bytes.Buffer
			descBuffer.WriteString("Terraform will delete the following infrastructure:\n")
			for _, target := range c.Meta.targets {
				descBuffer.WriteString("\t")
				descBuffer.WriteString(target)
				descBuffer.WriteString("\n")
			}
			descBuffer.WriteString("There is no undo. Only 'yes' will be accepted to confirm")
			desc = descBuffer.String()
		}

		v, err := c.UIInput().Input(&terraform.InputOpts{
			Id:          "destroy",
			Query:       "Do you really want to destroy?",
			Description: desc,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error asking for confirmation: %s", err))
			return 1
		}
		if v != "yes" {
			c.Ui.Output("Destroy cancelled.")
			return 1
		}
	}
	if !planned {
		if err := ctx.Input(c.InputMode()); err != nil {
			c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
			return 1
		}
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	// Plan if we haven't already
	if !planned {
		if refresh {
			if _, err := ctx.Refresh(); err != nil {
				c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
				return 1
			}
		}

		if _, err := ctx.Plan(); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error creating plan: %s", err))
			return 1
		}
	}

	// Setup the state hook for continuous state updates
	{
		state, err := c.State()
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error reading state: %s", err))
			return 1
		}

		stateHook.State = state
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
		go ctx.Stop()

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

	// Persist the state
	if state != nil {
		if err := c.Meta.PersistState(state); err != nil {
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
			multierror.Flatten(applyErr)))
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
			c.Meta.StateOutPath())))
	}

	if !c.Destroy {
		if outputs := outputsAsString(state); outputs != "" {
			c.Ui.Output(c.Colorize().Color(outputs))
		}
	}

	return 0
}

func (c *ApplyCommand) Help() string {
	if c.Destroy {
		return c.helpDestroy()
	}

	return c.helpApply()
}

func (c *ApplyCommand) Synopsis() string {
	if c.Destroy {
		return "Destroy Terraform-managed infrastructure"
	}

	return "Builds or changes infrastructure"
}

func (c *ApplyCommand) helpApply() string {
	helpText := `
Usage: terraform apply [options] [DIR]

  Builds or changes infrastructure according to Terraform configuration
  files in DIR.

  DIR can also be a SOURCE as given to the "init" command. In this case,
  apply behaves as though "init" was called followed by "apply". This only
  works for sources that aren't files, and only if the current working
  directory is empty of Terraform files. This is a shortcut for getting
  started.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -input=true            Ask for input for variables if not directly set.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of concurrent operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" is present, it will be
                         automatically loaded if this flag is not specified.


`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) helpDestroy() string {
	helpText := `
Usage: terraform destroy [options] [DIR]

  Destroy Terraform-managed infrastructure.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -force                 Don't ask for input for destroy confirmation.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of concurrent operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" is present, it will be
                         automatically loaded if this flag is not specified.


`
	return strings.TrimSpace(helpText)
}

func outputsAsString(state *terraform.State) string {
	if state == nil {
		return ""
	}

	outputs := state.RootModule().Outputs
	outputBuf := new(bytes.Buffer)
	if len(outputs) > 0 {
		outputBuf.WriteString("[reset][bold][green]\nOutputs:\n\n")

		// Output the outputs in alphabetical order
		keyLen := 0
		keys := make([]string, 0, len(outputs))
		for key, _ := range outputs {
			keys = append(keys, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := outputs[k]

			outputBuf.WriteString(fmt.Sprintf(
				"  %s%s = %s\n",
				k,
				strings.Repeat(" ", keyLen-len(k)),
				v))
		}
	}

	return strings.TrimSpace(outputBuf.String())
}
