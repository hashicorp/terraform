package command

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config/module"
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
	var refresh bool
	var statePath, stateOutPath, backupPath string

	args = c.Meta.process(args, true)

	cmdName := "apply"
	if c.Destroy {
		cmdName = "destroy"
	}

	cmdFlags := c.Meta.flagSet(cmdName)
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	cmdFlags.StringVar(&statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&backupPath, "backup", "", "path")
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
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The apply command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		configPath = pwd
	}

	// Prepare the extra hooks to count resources
	countHook := new(CountHook)
	c.Meta.extraHooks = []terraform.Hook{countHook}

	// If we don't specify an output path, default to out normal state
	// path.
	if stateOutPath == "" {
		stateOutPath = statePath
	}

	// If we don't specify a backup path, default to state out with
	// the extension
	if backupPath == "" {
		backupPath = stateOutPath + DefaultBackupExtention
	}

	if !c.Destroy {
		// Do a detect to determine if we need to do an init + apply.
		if detected, err := module.Detect(configPath, pwd); err != nil {
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
		Path:      configPath,
		StatePath: statePath,
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
	if c.InputEnabled() {
		if c.Destroy {
			v, err := c.UIInput().Input(&terraform.InputOpts{
				Id:    "destroy",
				Query: "Do you really want to destroy?",
				Description: "Terraform will delete all your manage infrastructure.\n" +
					"There is no undo. Only 'yes' will be accepted to confirm.",
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
			if err := ctx.Input(); err != nil {
				c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
				return 1
			}
		}
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	// Create a backup of the state before updating
	if backupPath != "-" && c.state != nil {
		log.Printf("[INFO] Writing backup state to: %s", backupPath)
		f, err := os.Create(backupPath)
		if err == nil {
			err = terraform.WriteState(c.state, f)
			f.Close()
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing backup state file: %s", err))
			return 1
		}
	}

	// Plan if we haven't already
	if !planned {
		if refresh {
			if _, err := ctx.Refresh(); err != nil {
				c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
				return 1
			}
		}

		var opts terraform.PlanOpts
		if c.Destroy {
			opts.Destroy = true
		}

		if _, err := ctx.Plan(&opts); err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error creating plan: %s", err))
			return 1
		}
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
	var outputs map[string]string
	if state != nil {
		outputs = state.RootModule().Outputs
	}
	if len(outputs) > 0 {
		outputBuf := new(bytes.Buffer)
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

		c.Ui.Output(c.Colorize().Color(
			strings.TrimSpace(outputBuf.String())))
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

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

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

  -input=true            Ask for input for destroy confirmation.

  -no-color              If specified, output won't contain any color.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" is present, it will be
                         automatically loaded if this flag is not specified.


`
	return strings.TrimSpace(helpText)
}
