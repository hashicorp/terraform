// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type NodeTestRun struct {
	file   *moduletest.File
	run    *moduletest.Run
	config *configs.Config

	Module addrs.Module
}

func (n *NodeTestRun) Run() *moduletest.Run {
	return n.run
}

func (n *NodeTestRun) File() *moduletest.File {
	return n.file
}

// GraphNodeReferencer
func (n *NodeTestRun) References() []*addrs.Reference {
	var result []*addrs.Reference
	// If we have a config then we prefer to use that.
	if c := n.run.Config; c != nil {
		refs, _ := n.run.GetReferences()
		result = append(result, refs...)
	}
	return result
}

// GraphNodeModulePath
func (n *NodeTestRun) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *NodeTestRun) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.run.Addr()}
}

func (n *NodeTestRun) Execute(testCtx *hcltest.TestContext, g *terraform.Graph) tfdiags.Diagnostics {
	// relevantVariables contains the variables that are of interest to this
	// run block. This is a combination of the variables declared within the
	// configuration for this run block, and the variables referenced by the
	// run block assertions.
	relevantVariables := make(map[string]bool)

	// First, we'll check to see which variables the run block assertions
	// reference.
	runRefs, diags := n.run.GetReferences()
	if diags.HasErrors() {
		return diags
	}
	for _, reference := range runRefs {
		if addr, ok := reference.Subject.(addrs.InputVariable); ok {
			relevantVariables[addr.Name] = true
		}
	}

	// And check to see which variables the run block configuration references.
	for name := range n.config.Module.Variables {
		relevantVariables[name] = true
	}

	// Now we'll get the values for all of these variables.
	variables := make(map[string]*terraform.InputValue)
	for name := range relevantVariables {
		value, diags := testCtx.GetGlobalVariable(name)
		if diags.HasErrors() {
			return diags
		}
		if value != nil {
			variables[name] = value
			continue
		}

		// If the variable wasn't a global variable, it might be a file variable.
		value, diags = testCtx.GetFileVariable(name)
		if diags.HasErrors() {
			return diags
		}

		if value != nil && value.Value.Type() != cty.DynamicPseudoType {
			variables[name] = value
			continue
		}

		// If the variable wasn't a file variable, it might be a run variable.
		value, diags = testCtx.GetRunVariable(n.run.Name, name)
		if diags.HasErrors() {
			return diags
		}
		if value != nil {
			variables[name] = value
			continue
		}

		// If the variable wasn't a run variable, it might be a config variable.
		value, diags = testCtx.GetConfigVariable(n.config.Module.SourceDir, name)
		if diags.HasErrors() {
			return diags
		}
		if value != nil {
			variables[name] = value
			continue
		}

	}

	for name := range relevantVariables {
		if _, exists := variables[name]; !exists {
			n.run.Status = moduletest.Error
			break
		}
	}

	return nil
}
