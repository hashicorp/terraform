// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeConfigVariable is a node that represents a variable in a terraform configuration.
// This node is created for each run, because the final value of the variable can be different
// depending on the file or run variables.
type nodeConfigVariable struct {
	Addr     addrs.InputVariable
	variable *configs.Variable

	run    *moduletest.Run
	config *configs.Config

	//Remove
	Module addrs.Module
}

var (
	_ terraform.GraphNodeReferenceable = (*nodeConfigVariable)(nil)
)

func (n *nodeConfigVariable) Name() string {
	return fmt.Sprintf("%s.%s (config,r=%s)", n.Module, n.Addr.Name, n.run.Name)
}

// GraphNodeModulePath
func (n *nodeConfigVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeConfigVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// TestGraphNodeExecutable
func (n *nodeConfigVariable) Execute(testCtx *hcltest.VariableContext, g *terraform.Graph) tfdiags.Diagnostics {
	// check if it is in the global or file variables first
	if variable, _ := testCtx.GetRunVariable(n.run.Name, n.variable.Name); variable != nil {
		return nil
	}
	if variable, _ := testCtx.GetFileVariable(n.variable.Name); variable != nil {
		return nil
	}
	if variable, _ := testCtx.GetGlobalVariable(n.variable.Name); variable != nil {
		return nil
	}

	// not found in global or file variables, so it must be a config variable
	// If it is optional, we're going to give these variables a value. They'll be
	// processed by the Terraform graph and provided a default value later
	// if they have one.
	testCtx.SetConfigVariable(n.config.Module, n.Addr.Name, n.variable)
	return nil
}
