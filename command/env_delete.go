package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/mitchellh/cli"

	clistate "github.com/hashicorp/terraform/command/state"
)

type EnvDeleteCommand struct {
	Meta
}

func (c *EnvDeleteCommand) Run(args []string) int {
	args = c.Meta.process(args, true)

	force := false
	cmdFlags := c.Meta.flagSet("env")
	cmdFlags.BoolVar(&force, "force", false, "force removal of a non-empty environment")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("expected NAME.\n")
		return cli.RunResultHelp
	}

	delEnv := args[0]

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

	states, current, err := multi.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	exists := false
	for _, s := range states {
		if delEnv == s {
			exists = true
			break
		}
	}

	if !exists {
		c.Ui.Error(fmt.Sprintf(envDoesNotExist, delEnv))
		return 1
	}

	// In order to check if the state being deleted is empty, we need to change
	// to that state and load it.
	if current != delEnv {
		if err := multi.ChangeState(delEnv); err != nil {
			c.Ui.Error(err.Error())
			return 1
		}

		// always try to change back after
		defer func() {
			if err := multi.ChangeState(current); err != nil {
				c.Ui.Error(err.Error())
			}
		}()
	}

	// we need the actual state to see if it's empty
	sMgr, err := b.State()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := sMgr.RefreshState(); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	empty := sMgr.State().Empty()

	if !empty && !force {
		c.Ui.Error(fmt.Sprintf(envNotEmpty, delEnv))
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

	err = multi.DeleteState(delEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envDeleted, delEnv),
		),
	)

	if !empty {
		c.Ui.Output(
			c.Colorize().Color(
				fmt.Sprintf(envWarnNotEmpty, delEnv),
			),
		)
	}

	return 0
}
func (c *EnvDeleteCommand) Help() string {
	helpText := `
Usage: terraform env delete [OPTIONS] NAME

  Delete a Terraform environment


Options:

    -force    remove a non-empty environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvDeleteCommand) Synopsis() string {
	return "Delete an environment"
}
