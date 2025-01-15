// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// nodeRunVariable is the placeholder for an variable that has not yet had
// its module path expanded.
type nodeRunVariable struct {
	Addr   addrs.InputVariable
	run    *moduletest.Run
	config *configs.Config
	Expr   hcl.Expression

	Module addrs.Module
}

var (
	_ terraform.GraphNodeReferenceable = (*nodeRunVariable)(nil)
	_ terraform.GraphNodeReferencer    = (*nodeRunVariable)(nil)
)

func (n *nodeRunVariable) Name() string {
	return fmt.Sprintf("%s.%s(run.%s)", n.Module, n.Addr.Name, n.run.Name)
}

// GraphNodeModulePath
func (n *nodeRunVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeRunVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeReferencer
func (n *nodeRunVariable) References() []*addrs.Reference {
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	return refs
}

// TestGraphNodeExecutable
func (n *nodeRunVariable) Execute(testCtx *hcltest.VariableContext, g *terraform.Graph) tfdiags.Diagnostics {

	//TODO: Do this only once
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

	// If we're testing a specific configuration, we need to use that
	config := n.config
	if n.run.Config.ConfigUnderTest != nil {
		config = n.run.Config.ConfigUnderTest
	}

	// And check to see which variables the run block configuration references.
	for name := range config.Module.Variables {
		relevantVariables[name] = true
	}

	requiredValues := make(map[string]cty.Value)
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	for _, ref := range refs {
		if addr, ok := ref.Subject.(addrs.InputVariable); ok {
			value, valueDiags := testCtx.GetFileVariable(addr.Name)
			diags = diags.Append(valueDiags)
			if value != nil {
				requiredValues[addr.Name] = value.Value
				continue
			}

			// Otherwise, it might be a global variable.
			value, valueDiags = testCtx.GetGlobalVariable(addr.Name)
			diags = diags.Append(valueDiags)
			if value != nil {
				requiredValues[addr.Name] = value.Value
				continue
			}
		}
	}
	diags = diags.Append(refDiags)

	ctx, ctxDiags := hcltest.EvalContext(hcltest.TargetRunBlock, map[string]hcl.Expression{n.Addr.Name: n.Expr}, requiredValues, testCtx.RunOutputs)
	diags = diags.Append(ctxDiags)

	value := cty.DynamicVal
	if !ctxDiags.HasErrors() {
		var valueDiags hcl.Diagnostics
		value, valueDiags = n.Expr.Value(ctx)
		diags = diags.Append(valueDiags)
	}

	// We do this late on so we still validate whatever it was that the user
	// wrote in the variable expression. But, we don't want to actually use
	// it if it's not actually relevant.
	if _, exists := relevantVariables[n.Addr.Name]; !exists {
		// Do not display warnings during cleanup2 phase
		// if includeWarnings { // TODO
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Value for undeclared variable",
			Detail:   fmt.Sprintf("The module under test does not declare a variable named %q, but it is declared in run block %q.", n.Addr.Name, n.run.Name),
			Subject:  n.Expr.Range().Ptr(),
		})
		// }
		testCtx.SetRunVariable(n.run.Name, n.Addr.Name, hcltest.VariableWithDiag{
			Value: &terraform.InputValue{
				Value:       value,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(n.Expr.Range()),
			},
			Diags: diags,
		})
		return nil
	}

	inputValue := &terraform.InputValue{
		Value:       value,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(n.Expr.Range()),
	}

	testCtx.SetRunVariable(n.run.Name, n.Addr.Name, hcltest.VariableWithDiag{
		Value: inputValue,
		Diags: diags,
	})
	return nil
}
