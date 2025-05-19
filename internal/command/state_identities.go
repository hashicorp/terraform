// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/cli"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateIdentitiesCommand is a Command implementation that lists the resource identities
// within a state file.
type StateIdentitiesCommand struct {
	Meta
	StateMeta
}

func (c *StateIdentitiesCommand) Run(args []string) int {
	args = c.Meta.process(args)
	var statePath string
	var jsonOutput bool
	cmdFlags := c.Meta.defaultFlagSet("state identities")
	cmdFlags.StringVar(&statePath, "state", "", "path")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
	lookupId := cmdFlags.String("id", "", "Restrict output to paths with a resource having the specified ID.")
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return cli.RunResultHelp
	}
	args = cmdFlags.Args()

	if !jsonOutput {
		c.Ui.Error(
			"The `terraform state identities` command requires the `-json` flag.\n")
		cmdFlags.Usage()
		return 1
	}

	if statePath != "" {
		c.Meta.statePath = statePath
	}

	// Load the backend
	b, backendDiags := c.Backend(nil)
	if backendDiags.HasErrors() {
		c.showDiagnostics(backendDiags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the state
	env, err := c.Workspace()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error selecting workspace: %s", err))
		return 1
	}
	stateMgr, err := b.StateMgr(env)
	if err != nil {
		c.Ui.Error(fmt.Sprintf(errStateLoadingState, err))
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		c.Ui.Error(errStateNotFound)
		return 1
	}

	var addrs []addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics
	if len(args) == 0 {
		addrs, diags = c.lookupAllResourceInstanceAddrs(state)
	} else {
		addrs, diags = c.lookupResourceInstanceAddrs(state, args...)
	}
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	output := make(map[string]any)
	for _, addr := range addrs {
		// If the resource exists but identity is nil, skip it, as it is not required to be present
		if is := state.ResourceInstance(addr); is != nil && is.Current.IdentityJSON != nil {
			if *lookupId == "" || *lookupId == states.LegacyInstanceObjectID(is.Current) {
				var rawIdentity map[string]any
				if err := json.Unmarshal(is.Current.IdentityJSON, &rawIdentity); err != nil {
					c.Ui.Error(fmt.Sprintf("Failed to unmarshal identity JSON: %s", err))
					return 1
				}
				output[addr.String()] = rawIdentity
			}
		}
	}

	outputJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to marshal output JSON: %s", err))
		return 1
	}

	c.Ui.Output(string(outputJSON))
	c.showDiagnostics(diags)

	return 0
}

func (c *StateIdentitiesCommand) Help() string {
	helpText := `
Usage: terraform [global options] state identities [options] -json [address...]

  List the json format of the identities of resources in the Terraform state.

  This command lists the identities of resource instances in the Terraform state in json format.
  The address argument can be used to filter the instances by resource or module. If
  no pattern is given, identities for all resource instances are listed.

  The addresses must either be module addresses or absolute resource
  addresses, such as:
      aws_instance.example
      module.example
      module.example.module.child
      module.example.aws_instance.example

  An error will be returned if any of the resources or modules given as
  filter addresses do not exist in the state.

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default, Terraform
                      will consult the state of the currently-selected
                      workspace.

  -id=ID              Filters the results to include only instances whose
                      resource types have an attribute named "id" whose value
                      equals the given id string.

`
	return strings.TrimSpace(helpText)
}

func (c *StateIdentitiesCommand) Synopsis() string {
	return "List the identities of resources in the state"
}
