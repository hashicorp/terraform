package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"

	clistate "github.com/hashicorp/terraform/command/state"
)

type EnvNewCommand struct {
	Meta
}

func (c *EnvNewCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	statePath := ""

	cmdFlags := c.Meta.flagSet("env new")
	cmdFlags.StringVar(&statePath, "state", "", "terraform state file")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("expected NAME.\n")
		return cli.RunResultHelp
	}

	newEnv := args[0]

	// Load the backend
	b, err := c.Backend(nil)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	multi, ok := b.(backend.MultiState)
	if !ok {
		c.Ui.Error(envNotSupported)
		return 1
	}

	states, _, err := multi.States()
	for _, s := range states {
		if newEnv == s {
			c.Ui.Error(fmt.Sprintf(envExists, newEnv))
			return 1
		}
	}

	err = multi.ChangeState(newEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envCreated, newEnv),
		),
	)

	if statePath == "" {
		// if we're not loading a state, then we're done
		return 0
	}

	// load the new Backend state
	sMgr, err := b.State()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Lock the state if we can
	lockInfo := state.NewLockInfo()
	lockInfo.Operation = "env new"
	lockID, err := clistate.Lock(sMgr, lockInfo, c.Ui, c.Colorize())
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
		return 1
	}
	defer clistate.Unlock(sMgr, lockID, c.Ui, c.Colorize())

	// read the existing state file
	stateFile, err := os.Open(statePath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	s, err := terraform.ReadState(stateFile)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// save the existing state in the new Backend.
	err = sMgr.WriteState(s)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *EnvNewCommand) Help() string {
	helpText := `
Usage: terraform env new [OPTIONS] NAME

  Create a new Terraform environment.


Options:

    -state=path    Copy an existing state file into the new environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvNewCommand) Synopsis() string {
	return "Create a new environment"
}
