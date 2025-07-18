// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/states/statemgr"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LockCommand is a cli.Command implementation that manually locks
// the state.
type LockCommand struct {
	Meta
}

func (c *LockCommand) Run(args []string) int {
	args = c.Meta.process(args)
	var force bool
	var operation string
	var info string
	
	cmdFlags := c.Meta.defaultFlagSet("force-lock")
	cmdFlags.BoolVar(&force, "force", false, "force lock without confirmation")
	cmdFlags.StringVar(&operation, "operation", "manual-lock", "operation description for the lock")
	cmdFlags.StringVar(&info, "info", "", "additional information to store with the lock")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 0 {
		c.Ui.Error("force-lock command does not accept any arguments")
		return cli.RunResultHelp
	}

	// assume everything is initialized. The user can manually init if this is
	// required.
	configPath, err := ModulePath(args)
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

	// This is a write operation with respect to the state lock
	c.ignoreRemoteVersionConflict(b)

	env, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	_, isLocal := stateMgr.(*statemgr.Filesystem)

	if !force {
		if isLocal {
			c.Ui.Error("Local state cannot be locked by another process")
			return 1
		}

		desc := "Terraform will acquire a lock on the remote state.\n" +
			"This will prevent other local Terraform commands from modifying this state.\n" +
			"Only 'yes' will be accepted to confirm."

		v, err := c.UIInput().Input(context.Background(), &terraform.InputOpts{
			Id:          "force-lock",
			Query:       "Do you really want to force-lock?",
			Description: desc,
		})
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error asking for confirmation: %s", err))
			return 1
		}
		if v != "yes" {
			c.Ui.Output("force-lock cancelled.")
			return 1
		}
	}

	// Create lock info
	lockInfo := statemgr.NewLockInfo()
	lockInfo.Operation = operation
	lockInfo.Info = info

	lockID, err := stateMgr.Lock(lockInfo)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to lock state: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(outputLockSuccess, lockID, lockID)))
	return 0
}

func (c *LockCommand) Help() string {
	helpText := `
Usage: terraform [global options] force-lock

  Manually lock the state for the defined configuration.

  This will not modify your infrastructure. This command acquires a lock on the
  state for the current workspace. The behavior of this lock is dependent
  on the backend being used. Local state files cannot be locked by another
  process.

  The lock ID will be displayed after successful locking. Use this ID with
  'terraform force-unlock' to release the lock.

Options:

  -force                 Don't ask for input for lock confirmation.
  -operation=<string>    Operation description for the lock (default: "manual-lock").
  -info=<string>         Additional information to store with the lock.
`
	return strings.TrimSpace(helpText)
}

func (c *LockCommand) Synopsis() string {
	return "Acquire a lock on the current workspace"
}

const outputLockSuccess = `
[reset][bold][green]Terraform state has been successfully locked![reset][green]

Lock ID: %s

The state has been locked. Use 'terraform force-unlock %s' to release this lock.
Other Terraform commands will not be able to obtain a lock on the remote state
until this lock is released.
`