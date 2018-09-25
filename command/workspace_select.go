package command

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

type WorkspaceSelectCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceSelectCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	envCommandShowWarning(c.Ui, c.LegacyName)

	statePath := ""
	var createWorkspace bool

	cmdFlags := c.Meta.flagSet("workspace select")
	cmdFlags.BoolVar(&createWorkspace, "create", false, "create workspace")
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

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	conf, err := c.Config(configPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load root config module: %s", err))
	}

	current, isOverridden := c.WorkspaceOverridden()
	if isOverridden {
		c.Ui.Error(envIsOverriddenSelectError)
		return 1
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
	})

	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	name := args[0]
	if !validWorkspaceName(name) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, name))
		return 1
	}

	states, err := b.States()
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if name == current {
		// already using this workspace
		return 0
	}

	found := false
	for _, s := range states {
		if name == s {
			found = true
			break
		}
	}

	if !found {

		// we don't want to create a workspace if it does not exist
		if !createWorkspace {
			c.Ui.Error(fmt.Sprintf(envDoesNotExist, name))
			return 1
		} else {
			// we do want to create a workspace if it does not exist

			// new workspace name
			_, err = b.State(name)
			if err != nil {
				c.Ui.Error(err.Error())
			}

			// now set the current workspace locally
			if err := c.SetWorkspace(name); err != nil {
				c.Ui.Error(fmt.Sprintf("Error selecting new workspace: %s", err))
				return 1
			}

			// tell the client we've created the workspace
			c.Ui.Output(
				c.Colorize().Color(
					fmt.Sprintf(strings.TrimSpace(envCreated), name),
				),
			)

			// if we aren't loading a state file, bail early
			if statePath == "" {
				return 0
			}

			// load the new backend's state
			sMgr, err := b.State(name)
			if err != nil {
				c.Ui.Error(err.Error())
				return 1
			}

			if c.stateLock {
				stateLocker := clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
				if err := stateLocker.Lock(sMgr, "workspace_delete"); err != nil {
					c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
					return 1
				}
				defer stateLocker.Unlock(nil)
			}

			// read existing state file
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

			// save the existing state in the new backend
			if err = sMgr.WriteState(s); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}
			if err = sMgr.PersistState(); err != nil {
				c.Ui.Error(err.Error())
				return 1
			}

			// we've successfully switched to a new workspace
			return 0

		}
	}

	err = c.SetWorkspace(name)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	c.Ui.Output(
		c.Colorize().Color(
			fmt.Sprintf(envChanged, name),
		),
	)

	return 0
}

func (c *WorkspaceSelectCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		complete.PredictNothing, // the "select" subcommand itself (already matched)
		c.completePredictWorkspaceName(),
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceSelectCommand) AutocompleteFlags() complete.Flags {
	return nil
}

func (c *WorkspaceSelectCommand) Help() string {
	helpText := `
Usage: terraform workspace select [OPTIONS] NAME [DIR]

  Select a different Terraform workspace.


Options:

    -create         Create the workspace if it does not exist
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceSelectCommand) Synopsis() string {
	return "Select a workspace"
}
