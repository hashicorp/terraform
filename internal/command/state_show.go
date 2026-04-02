// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

// StateShowCommand is a Command implementation that shows a single resource.
type StateShowCommand struct {
	Meta
	StateMeta
	viewType arguments.ViewType
}

func (c *StateShowCommand) Run(args []string) int {
	// Parse and apply global view arguments
	common, args := arguments.ParseView(args)
	c.View.Configure(common)

	parsedArgs, diags := arguments.ParseStateShow(args)
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		c.View.HelpPrompt("state show")
		return 1
	}

	c.Meta.statePath = parsedArgs.StatePath
	c.viewType = parsedArgs.ViewType
	view := views.NewShow(parsedArgs.ViewType, c.View)

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading plugin path: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, diags := c.backend(".", c.viewType)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backendrun.Local)
	if !ok {
		diags = diags.Append(ErrUnsupportedLocalOp)
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Check if the address can be parsed
	addr, addrDiags := addrs.ParseAbsResourceInstanceStr(parsedArgs.Address)
	if addrDiags.HasErrors() {
		diags = diags.Append(fmt.Sprintf(errParsingAddress, parsedArgs.Address))
		view.Diagnostics(diags)
		return 1
	}

	// We expect the config dir to always be the cwd
	cwd, err := os.Getwd()
	if err != nil {
		diags = diags.Append(fmt.Sprintf("Error getting cwd: %s\n", err))
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation (required to get the schemas)
	opReq := c.Operation(b, c.viewType)
	opReq.AllowUnsetVariables = true
	opReq.ConfigDir = cwd

	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Sprintf("Error initializing config loader: %s\n", err))
		view.Diagnostics(diags)
		return 1
	}

	// Get the context (required to get the schemas)
	lr, _, ctxDiags := local.LocalRun(opReq)
	if ctxDiags.HasErrors() {
		view.Diagnostics(ctxDiags)
		return 1
	}

	// Get the schemas from the context
	schemas, diags := lr.Core.Schemas(lr.Config, lr.InputState)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Get the state
	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(fmt.Sprintf("Error selecting workspace: %s\n", err))
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
		diags = diags.Append(fmt.Errorf("Failed to refresh state: %s\n", err))
		view.Diagnostics(diags)
		return 1
	}

	state := stateMgr.State()
	if state == nil {
		diags = diags.Append(errors.New(errStateNotFound))
		view.Diagnostics(diags)
		return 1
	}

	is := state.ResourceInstance(addr)
	if !is.HasCurrent() {
		diags = diags.Append(errors.New(errNoInstanceFound))
		view.Diagnostics(diags)
		return 1
	}

	// check if the resource has a configured provider, otherwise this will use the default provider
	rs := state.Resource(addr.ContainingResource())
	absPc := addrs.AbsProviderConfig{
		Provider: rs.ProviderConfig.Provider,
		Alias:    rs.ProviderConfig.Alias,
		Module:   addrs.RootModule,
	}
	singleInstance := states.NewState()
	singleInstance.EnsureModule(addr.Module).SetResourceInstanceCurrent(
		addr.Resource,
		is.Current,
		absPc,
	)

	mockFile := statefile.New(singleInstance, "", 0)
	root, outputs, err := jsonstate.MarshalForRenderer(mockFile, schemas)
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to marshal state to json: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	jstate := jsonformat.State{
		StateFormatVersion:    jsonstate.FormatVersion,
		ProviderFormatVersion: jsonprovider.FormatVersion,
		RootModule:            root,
		RootModuleOutputs:     outputs,
		ProviderSchemas:       jsonprovider.MarshalForRenderer(schemas),
	}

	if c.viewType == arguments.ViewHuman {
		renderer := jsonformat.Renderer{
			Streams:             c.Streams,
			Colorize:            c.Colorize(),
			RunningInAutomation: c.RunningInAutomation,
		}
		renderer.RenderHumanState(jstate)
	} else {
		view.DisplayResourceInstanceState(rs, addr, schemas)
	}

	return 0
}

func (c *StateShowCommand) Help() string {
	helpText := `
Usage: terraform [global options] state show [options] ADDRESS

  Shows the attributes of a resource in the Terraform state.

  This command shows the attributes of a single resource in the Terraform
  state. The address argument must be used to specify a single resource.
  You can view the list of available resources with "terraform state list".

Options:

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

`
	return strings.TrimSpace(helpText)
}

func (c *StateShowCommand) Synopsis() string {
	return "Show a resource in the state"
}

const errNoInstanceFound = `No instance found for the given address!

This command requires that the address references one specific instance.
To view the available instances, use "terraform state list". Please modify
the address to reference a specific instance.`

const errParsingAddress = `Error parsing instance address: %s

This command requires that the address references one specific instance.
To view the available instances, use "terraform state list". Please modify
the address to reference a specific instance.`
