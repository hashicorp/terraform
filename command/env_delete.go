package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/state"
	"github.com/mitchellh/cli"
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
	if len(args) == 0 {
		c.Ui.Error("expected NAME.\n")
		return cli.RunResultHelp
	}

	delEnv := args[0]

	if !validEnvName(delEnv) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, delEnv))
		return 1
	}

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{ConfigPath: configPath})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	states, err := b.States()
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
		c.Ui.Error(fmt.Sprintf(strings.TrimSpace(envDoesNotExist), delEnv))
		return 1
	}

	if delEnv == c.Env() {
		c.Ui.Error(fmt.Sprintf(strings.TrimSpace(envDelCurrent), delEnv))
		return 1
	}

	// we need the actual state to see if it's empty
	sMgr, err := b.State(delEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := sMgr.RefreshState(); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	hasResources := sMgr.State().HasResources()

	if hasResources && !force {
		c.Ui.Error(fmt.Sprintf(strings.TrimSpace(envNotEmpty), delEnv))
		return 1
	}

	if c.stateLock {
		lockCtx, cancel := context.WithTimeout(context.Background(), c.stateLockTimeout)
		defer cancel()

		// Lock the state if we can
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "env delete"
		lockID, err := clistate.Lock(lockCtx, sMgr, lockInfo, c.Ui, c.Colorize())
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
		defer clistate.Unlock(sMgr, lockID, c.Ui, c.Colorize())
	}

	err = b.DeleteState(delEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envDeleted, delEnv),
		),
	)

	if hasResources {
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
Usage: terraform env delete [OPTIONS] NAME [DIR]

  Delete a Terraform environment


Options:

    -force    remove a non-empty environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvDeleteCommand) Synopsis() string {
	return "Delete an environment"
}
