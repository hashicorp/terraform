// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

func (n *nodeConfigVariable) expandsInstances() {}

func (n *nodeConfigVariable) temporaryValue() bool {
	return true
}

func (n *nodeConfigVariable) Name() string {
	return fmt.Sprintf("%s.%s (expand)", n.Module, n.Addr.String())
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
func (n *nodeConfigVariable) Execute(testCtx *hcltest.TestContext, g *terraform.Graph) tfdiags.Diagnostics {
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
	var diags hcl.Diagnostics
	var value *terraform.InputValue
	if n.variable.Required() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No value for required variable",
			Detail: fmt.Sprintf("The module under test has a required variable %q with no set value. Use a -var or -var-file command line argument or add this variable into a \"variables\" block within the test file or run block.",
				n.variable.Name),
			Subject: n.variable.DeclRange.Ptr(),
		})

		value = &terraform.InputValue{
			Value:       cty.DynamicVal,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(n.variable.DeclRange),
		}
	} else {
		value = &terraform.InputValue{
			Value:       cty.NilVal,
			SourceType:  terraform.ValueFromConfig,
			SourceRange: tfdiags.SourceRangeFromHCL(n.variable.DeclRange),
		}
	}
	key := n.config.Module.SourceDir
	tfDiags := testCtx.SetConfigVariable(n.Addr.Name, value)
	mp, ok := testCtx.ConfigVariables2[key]
	if !ok {
		mp = make(terraform.InputValues)
	}
	mp[n.Addr.Name] = value
	testCtx.ConfigVariables2[key] = mp
	return tfDiags.Append(diags)
}
