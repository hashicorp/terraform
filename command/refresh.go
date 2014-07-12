package command

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	ContextOpts *terraform.ContextOpts
	Ui          cli.Ui
}

func (c *RefreshCommand) Run(args []string) int {
	var statePath, stateOutPath string

	cmdFlags := flag.NewFlagSet("refresh", flag.ContinueOnError)
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

	// Build the context based on the arguments given
	c.ContextOpts.Hooks = append(c.ContextOpts.Hooks, &UiHook{Ui: c.Ui})
	ctx, err := ContextArg(configPath, statePath, c.ContextOpts)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}

	state, err := ctx.Refresh()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
		return 1
	}

	log.Printf("[INFO] Writing state output to: %s", stateOutPath)
	f, err := os.Create(stateOutPath)
	if err == nil {
		defer f.Close()
		err = terraform.WriteState(state, f)
	}
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	return 0
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options] [dir]

  Update the state file of your infrastructure with metadata that matches
  the physical resources they are tracking.

  This will not modify your infrastructure, but it can modify your
  state file to update metadata. This metadata might cause new changes
  to occur when you generate a plan or call apply next.

Options:

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

`
	return strings.TrimSpace(helpText)
}

func (c *RefreshCommand) Synopsis() string {
	return "Refresh the local state of your infrastructure"
}
