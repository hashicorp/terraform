package command

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
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
	if len(args) == 0 {
		c.Ui.Error("Expected a single argument: NAME.\n")
		return cli.RunResultHelp
	}

	newEnv := args[0]

	if !validEnvName(newEnv) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, newEnv))
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
	for _, s := range states {
		if newEnv == s {
			c.Ui.Error(fmt.Sprintf(envExists, newEnv))
			return 1
		}
	}

	_, err = b.State(newEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// now save the current env locally
	if err := c.SetEnv(newEnv); err != nil {
		c.Ui.Error(fmt.Sprintf("error saving new environment name: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		strings.TrimSpace(envCreated), newEnv)))

	if statePath == "" {
		// if we're not loading a state, then we're done
		return 0
	}

	// load the new Backend state
	sMgr, err := b.State(newEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if c.stateLock {
		lockCtx, cancel := context.WithTimeout(context.Background(), c.stateLockTimeout)
		defer cancel()

		// Lock the state if we can
		lockInfo := state.NewLockInfo()
		lockInfo.Operation = "env new"
		lockID, err := clistate.Lock(lockCtx, sMgr, lockInfo, c.Ui, c.Colorize())
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
		defer clistate.Unlock(sMgr, lockID, c.Ui, c.Colorize())
	}

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
	err = sMgr.PersistState()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	return 0
}

func (c *EnvNewCommand) Help() string {
	helpText := `
Usage: terraform env new [OPTIONS] NAME [DIR]

  Create a new Terraform environment.


Options:

    -state=path    Copy an existing state file into the new environment.
`
	return strings.TrimSpace(helpText)
}

func (c *EnvNewCommand) Synopsis() string {
	return "Create a new environment"
}
