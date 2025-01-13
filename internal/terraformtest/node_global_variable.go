// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeGlobalVariable is a node that represents a variable that comes from the
// global variables of the configuration, either from the CLI or from the
// tfvars file.
type nodeGlobalVariable struct {
	Addr        addrs.InputVariable
	unparsed    backendrun.UnparsedVariableValue
	parsingMode configs.VariableParsingMode
	config      *configs.Config

	//Remove
	Module addrs.Module
}

var (
	_ terraform.GraphNodeReferenceable = (*nodeGlobalVariable)(nil)
)

func (n *nodeGlobalVariable) Name() string {
	return fmt.Sprintf("%s.%s (global)", n.Module, n.Addr.String())
}

// GraphNodeModulePath
func (n *nodeGlobalVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeGlobalVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// TestGraphNodeExecutable
func (n *nodeGlobalVariable) Execute(testCtx *hcltest.VariableContext, g *terraform.Graph) tfdiags.Diagnostics {
	value, diags := n.unparsed.ParseVariableValue(n.parsingMode)
	if diags.HasErrors() {
		// In this case, the variable exists but we couldn't parse it. We'll
		// return a usable value so that we don't compound errors later by
		// claiming a variable doesn't exist when it does. We also return the
		// diagnostics explaining the error which will be shown to the user.
		value = &terraform.InputValue{
			Value: cty.DynamicVal,
		}
	}
	testCtx.SetGlobalVariable(n.Addr.Name, value)
	return diags
}
