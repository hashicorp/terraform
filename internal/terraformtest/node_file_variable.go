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

// nodeFileVariable is the placeholder for an variable that has not yet had
// its module path expanded.
type nodeFileVariable struct {
	run    *moduletest.Run
	config *configs.Config
	Addr   addrs.InputVariable
	Expr   hcl.Expression

	//Remove
	Module addrs.Module
}

var (
	_ terraform.GraphNodeReferenceable = (*nodeFileVariable)(nil)
	_ terraform.GraphNodeReferencer    = (*nodeFileVariable)(nil)
)

func (n *nodeFileVariable) Name() string {
	return fmt.Sprintf("%s.%s (file)", n.Module, n.Addr.String())
}

// GraphNodeModulePath
func (n *nodeFileVariable) ModulePath() addrs.Module {
	return n.Module
}

// GraphNodeReferenceable
func (n *nodeFileVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// GraphNodeReferencer
func (n *nodeFileVariable) References() []*addrs.Reference {
	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	return refs
}

// TestGraphNodeExecutable
func (n *nodeFileVariable) Execute(testCtx *hcltest.VariableContext, g *terraform.Graph) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	availableVariables := make(map[string]cty.Value)
	value := &terraform.InputValue{
		Value: cty.DynamicVal,
	}
	// If we had referenced a global variable in the file variable, we need to
	// get it from the global variables store. e.g. `var.foo` is inside the file var
	refs, refDiags := langrefs.ReferencesInExpr(addrs.ParseRefFromTestingScope, n.Expr)
	for _, ref := range refs {
		if input, ok := ref.Subject.(addrs.InputVariable); ok {
			variable, variableDiags := testCtx.GetGlobalVariable(input.Name)
			diags = diags.Append(variableDiags)
			if variable != nil {
				availableVariables[input.Name] = variable.Value
			}
		}
	}
	diags = diags.Append(refDiags)

	if diags.HasErrors() {
		// There's no point trying to evaluate the variable as we know it will
		// fail. We'll just return a usable value so that we don't compound
		// errors later by claiming a variable doesn't exist when it does. We
		// also return the diagnostics explaining the error which will be shown
		// to the user.
		testCtx.SetFileVariable(n.Addr.Name, hcltest.VariableWithDiag{
			Value: value,
			Diags: diags,
		})
		return nil
	}

	ctx, ctxDiags := hcltest.EvalContext(hcltest.TargetFileVariable, map[string]hcl.Expression{n.Addr.Name: n.Expr}, availableVariables, nil)
	diags = diags.Append(ctxDiags)

	if ctxDiags.HasErrors() {
		// If we couldn't build the context, we won't actually process these
		// variables. Instead, we'll fill them with an empty value but still
		// make a note that the user did provide them.
		testCtx.SetFileVariable(n.Addr.Name, hcltest.VariableWithDiag{
			Value: value,
			Diags: ctxDiags,
		})
		return nil
	}

	ctyValue, valueDiags := n.Expr.Value(ctx)
	diags = diags.Append(valueDiags)
	if diags.HasErrors() {
		// In this case, the variable exists but we couldn't parse it. We'll
		// return a usable value so that we don't compound errors later by
		// claiming a variable doesn't exist when it does. We also return the
		// diagnostics explaining the error which will be shown to the user.
		ctyValue = cty.DynamicVal
	}

	value = &terraform.InputValue{
		Value:       ctyValue,
		SourceType:  terraform.ValueFromConfig,
		SourceRange: tfdiags.SourceRangeFromHCL(n.Expr.Range()),
	}
	testCtx.SetFileVariable(n.Addr.Name, hcltest.VariableWithDiag{
		Value: value,
		Diags: diags,
	})
	return nil
}
