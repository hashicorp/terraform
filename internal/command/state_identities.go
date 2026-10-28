// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StateIdentitiesCommand is a Command implementation that lists the resource identities
// within a state file.
type StateIdentitiesCommand struct {
	StateMeta
}

func (c *StateIdentitiesCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments
	args, argDiags := arguments.ParseStateIdentities(rawArgs)

	if args.StatePath != "" {
		c.Meta.statePath = args.StatePath
	}

	view := views.NewStateIdentities(args.ViewType, c.View)

	if argDiags.HasErrors() {
		view.Diagnostics(argDiags)
		return 1
	}

	// Load the backend
	b, diags := c.backend(".", args.ViewType)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the state
	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to select workspace",
			fmt.Sprintf("Error selecting workspace: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}
	stateMgr, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(sDiags)
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error loading the state",
			`Please ensure that your Terraform state exists and that you've
configured it properly. You can use the "-state" flag to point
Terraform at another state file.`,
		))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to refresh the state",
			fmt.Sprintf("Error refreshing the state: %s", err),
		))
		view.Diagnostics(diags)
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No state file was found",
			`State management commands require a state file. Run this command
in a directory where Terraform has been run or use the -state flag
to point the command to a specific state location.`,
		))
		view.Diagnostics(diags)
		return 1
	}

	var addrs []addrs.AbsResourceInstance
	if len(args.Addrs) == 0 {
		addrs, diags = c.lookupAllResourceInstanceAddrs(state)
	} else {
		addrs, diags = c.lookupResourceInstanceAddrs(state, args.Addrs...)
	}
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	identities := make(map[string]json.RawMessage)
	for _, addr := range addrs {
		// If the resource exists but identity is nil, skip it, as it is not required to be present
		if is := state.ResourceInstance(addr); is != nil && is.Current.IdentityJSON != nil {
			if args.ID == "" || args.ID == states.LegacyInstanceObjectID(is.Current) {
				identities[addr.String()] = is.Current.IdentityJSON
			}
		}
	}

	view.Identities(identities)

	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

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

  -id=ID              Filters the results to include only instances whose
                      resource types have an attribute named "id" whose value
                      equals the given id string.

  -json               Outputs the identities in JSON format. This flag is required.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default, Terraform
                      will consult the state of the currently-selected
                      workspace.

`
	return strings.TrimSpace(helpText)
}

func (c *StateIdentitiesCommand) Synopsis() string {
	return "List the identities of resources in the state"
}
