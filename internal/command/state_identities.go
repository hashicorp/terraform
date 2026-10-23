// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
)

// StateIdentitiesCommand is a Command implementation that lists the resource identities
// within a state file.
type StateIdentitiesCommand struct {
	StateMeta
}

func (c *StateIdentitiesCommand) Run(rawArgs []string) int {
	rawArgs = c.Meta.process(rawArgs)

	args, diags := arguments.ParseStateIdentities(rawArgs)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		return 1
	}

	if args.StatePath != "" {
		c.Meta.statePath = args.StatePath
	}

	view := views.NewStateIdentities(args.ViewType, c.View)

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
		diags = diags.Append(fmt.Errorf("Error selecting workspace: %s", err))
		view.Diagnostics(diags)
		return 1
	}
	stateMgr, sDiags := b.StateMgr(env)
	if sDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf(errStateLoadingState, sDiags.Err()))
		view.Diagnostics(diags)
		return 1
	}
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("Failed to load state: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		diags = diags.Append(errors.New(errStateNotFound))
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

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default, Terraform
                      will consult the state of the currently-selected
                      workspace.

  -id=ID              Filters the results to include only instances whose
                      resource types have an attribute named "id" whose value
                      equals the given id string.

  -json               Outputs the identities in JSON format. This flag is required.

`
	return strings.TrimSpace(helpText)
}

func (c *StateIdentitiesCommand) Synopsis() string {
	return "List the identities of resources in the state"
}
