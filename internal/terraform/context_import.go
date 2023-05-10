// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ImportOpts are used as the configuration for Import.
type ImportOpts struct {
	// Targets are the targets to import
	Targets []*ImportTarget

	// SetVariables are the variables set outside of the configuration,
	// such as on the command line, in variables files, etc.
	SetVariables InputValues
}

// ImportTarget is a single resource to import.
type ImportTarget struct {
	// Config is the original import block for this import. This might be null
	// if the import did not originate in config.
	Config *configs.Import

	// Addr is the address for the resource instance that the new object should
	// be imported into.
	Addr addrs.AbsResourceInstance

	// ID is the ID of the resource to import. This is resource-specific.
	ID string
}

// Import takes already-created external resources and brings them
// under Terraform management. Import requires the exact type, name, and ID
// of the resources to import.
//
// This operation is idempotent. If the requested resource is already
// imported, no changes are made to the state.
//
// Further, this operation also gracefully handles partial state. If during
// an import there is a failure, all previously imported resources remain
// imported.
func (c *Context) Import(config *configs.Config, prevRunState *states.State, opts *ImportOpts) (*states.State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Hold a lock since we can modify our own state here
	defer c.acquireRun("import")()

	// Don't modify our caller's state
	state := prevRunState.DeepCopy()

	log.Printf("[DEBUG] Building and walking import graph")

	variables := opts.SetVariables

	// Initialize our graph builder
	builder := &PlanGraphBuilder{
		ImportTargets:      opts.Targets,
		Config:             config,
		State:              state,
		RootVariableValues: variables,
		Plugins:            c.plugins,
		Operation:          walkImport,
	}

	// Build the graph
	graph, graphDiags := builder.Build(addrs.RootModuleInstance)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return state, diags
	}

	// Walk it
	walker, walkDiags := c.walk(graph, walkImport, &graphWalkOpts{
		Config:     config,
		InputState: state,
	})
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return state, diags
	}

	// Data sources which could not be read during the import plan will be
	// unknown. We need to strip those objects out so that the state can be
	// serialized.
	walker.State.RemovePlannedResourceInstanceObjects()

	newState := walker.State.Close()
	return newState, diags
}
