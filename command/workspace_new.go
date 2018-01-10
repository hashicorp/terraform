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
	"github.com/posener/complete"
)

type WorkspaceNewCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceNewCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	envCommandShowWarning(c.Ui, c.LegacyName)

	statePath := ""

	cmdFlags := c.Meta.flagSet("workspace new")
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

	if !validWorkspaceName(newEnv) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, newEnv))
		return 1
	}

	// You can't ask to create a workspace when you're overriding the
	// workspace name to be something different.
	if current, isOverridden := c.WorkspaceOverridden(); current != newEnv && isOverridden {
		c.Ui.Error(envIsOverriddenNewError)
		return 1
	}

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	conf, err := c.Config(configPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load root config module: %s", err))
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
	})

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	states, err := b.States()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to get configured named states: %s", err))
		return 1
	}
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

	// now set the current workspace locally
	if err := c.SetWorkspace(newEnv); err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting new workspace: %s", err))
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
		lockInfo.Operation = "workspace new"
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

func (c *WorkspaceNewCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		complete.PredictNothing, // the "new" subcommand itself (already matched)
		complete.PredictAnything,
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceNewCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-state": complete.PredictFiles("*.tfstate"),
	}
}

func (c *WorkspaceNewCommand) Help() string {
	helpText := `
Usage: terraform workspace new [OPTIONS] NAME [DIR]

  Create a new Terraform workspace.


Options:

    -state=path    Copy an existing state file into the new workspace.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceNewCommand) Synopsis() string {
	return "Create a new workspace"
}
