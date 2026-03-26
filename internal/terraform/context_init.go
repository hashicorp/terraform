// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type InitOpts struct {
	Walker configs.ModuleWalker

	// SetVariables are the raw values for root module variables as provided
	// by the user who is requesting the run, prior to any normalization or
	// substitution of defaults. See the documentation for the InputValue
	// type for more information on how to correctly populate this.
	SetVariables InputValues
}

func (c *Context) Init(rootMod *configs.Module, initOpts InitOpts) (*configs.Config, tfdiags.Diagnostics) {
	return c.init(rootMod, initOpts)
}

func (c *Context) init(rootMod *configs.Module, initOpts InitOpts) (*configs.Config, tfdiags.Diagnostics) {
	defer c.acquireRun("init")()
	var diags tfdiags.Diagnostics

	config := &configs.Config{
		Module:   rootMod,
		Path:     addrs.RootModule,
		Children: map[string]*configs.Config{},
	}
	config.Root = config

	graph, moreDiags := c.initGraph(config, initOpts)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	walker, walkDiags := c.walk(graph, walkInit, &graphWalkOpts{
		Config: config,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	return config, diags
}

func (c *Context) initGraph(config *configs.Config, initOpts InitOpts) (*Graph, tfdiags.Diagnostics) {
	graph, diags := (&InitGraphBuilder{
		Config:             config,
		RootVariableValues: initOpts.SetVariables,
		Walker:             initOpts.Walker,
	}).Build(addrs.RootModuleInstance)

	return graph, diags
}
