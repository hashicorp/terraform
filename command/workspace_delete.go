package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mitchellh/cli"
	"github.com/posener/complete"
)

type WorkspaceDeleteCommand struct {
	Meta
	LegacyName bool
}

func (c *WorkspaceDeleteCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	envCommandShowWarning(c.Ui, c.LegacyName)

	force := false
	cmdFlags := c.Meta.flagSet("workspace")
	cmdFlags.BoolVar(&force, "force", false, "force removal of a non-empty workspace")
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

	if !validWorkspaceName(delEnv) {
		c.Ui.Error(fmt.Sprintf(envInvalidName, delEnv))
		return 1
	}

	configPath, err := ModulePath(args[1:])
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
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

	if delEnv == c.Workspace() {
		c.Ui.Error(fmt.Sprintf(strings.TrimSpace(envDelCurrent), delEnv))
		return 1
	}

	// we need the actual state to see if it's empty
	sMgr, err := b.State(delEnv)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	var stateLocker clistate.Locker
	if c.stateLock {
		stateLocker = clistate.NewLocker(context.Background(), c.stateLockTimeout, c.Ui, c.Colorize())
		if err := stateLocker.Lock(sMgr, "workspace_delete"); err != nil {
			c.Ui.Error(fmt.Sprintf("Error locking state: %s", err))
			return 1
		}
	} else {
		stateLocker = clistate.NewNoopLocker()
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

	// We need to release the lock just before deleting the state, in case
	// the backend can't remove the resource while holding the lock. This
	// is currently true for Windows local files.
	//
	// TODO: While there is little safety in locking while deleting the
	// state, it might be nice to be able to coordinate processes around
	// state deletion, i.e. in a CI environment. Adding Delete() as a
	// required method of States would allow the removal of the resource to
	// be delegated from the Backend to the State itself.
	stateLocker.Unlock(nil)

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

func (c *WorkspaceDeleteCommand) AutocompleteArgs() complete.Predictor {
	return completePredictSequence{
		complete.PredictNothing, // the "select" subcommand itself (already matched)
		c.completePredictWorkspaceName(),
		complete.PredictDirs(""),
	}
}

func (c *WorkspaceDeleteCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-force": complete.PredictNothing,
	}
}

func (c *WorkspaceDeleteCommand) Help() string {
	helpText := `
Usage: terraform workspace delete [OPTIONS] NAME [DIR]

  Delete a Terraform workspace


Options:

    -force    remove a non-empty workspace.
`
	return strings.TrimSpace(helpText)
}

func (c *WorkspaceDeleteCommand) Synopsis() string {
	return "Delete a workspace"
}
