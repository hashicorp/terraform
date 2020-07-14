package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// NodeRootVariable represents a root variable input.
type NodeRootVariable struct {
	Addr   addrs.InputVariable
	Config *configs.Variable
}

var (
	_ GraphNodeModuleInstance = (*NodeRootVariable)(nil)
	_ GraphNodeReferenceable  = (*NodeRootVariable)(nil)
)

func (n *NodeRootVariable) Name() string {
	return n.Addr.String()
}

// GraphNodeModuleInstance
func (n *NodeRootVariable) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

func (n *NodeRootVariable) ModulePath() addrs.Module {
	return addrs.RootModule
}

// GraphNodeReferenceable
func (n *NodeRootVariable) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr}
}

// dag.GraphNodeDotter impl.
func (n *NodeRootVariable) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "note",
		},
	}
}

// GraphNodeExecable
func (n *NodeRootVariable) Execute(ctx EvalContext) tfdiags.Diagnostics {
	// We don't actually need to _evaluate_ a root module variable, because
	// its value is always constant and already stashed away in our EvalContext.
	// However, we might need to run some user-defined validation rules against
	// the value.

	if n.Config == nil || len(n.Config.Validations) == 0 {
		log.Printf("[TRACE] execVariableValidations: not active for %s, so skipping", n.Addr)
		return nil
	}

	var diags tfdiags.Diagnostics

	// Variable nodes evaluate in the parent module to where they were declared
	// because the value expression (n.Expr, if set) comes from the calling
	// "module" block in the parent module.
	//
	// Validation expressions are statically validated (during configuration
	// loading) to refer only to the variable being validated, so we can
	// bypass our usual evaluation machinery here and just produce a minimal
	// evaluation context containing just the required value, and thus avoid
	// the problem that ctx's evaluation functions refer to the wrong module.
	val := ctx.GetVariableValue(n.Addr.Absolute(n.Path()))
	hclCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"var": cty.ObjectVal(map[string]cty.Value{
				n.Config.Name: val,
			}),
		},
		Functions: ctx.EvaluationScope(nil, EvalDataForNoInstanceKey).Functions(),
	}

	for _, validation := range n.Config.Validations {
		const errInvalidCondition = "Invalid variable validation result"
		const errInvalidValue = "Invalid value for variable"

		result, moreDiags := validation.Condition.Value(hclCtx)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition expression failed: %s", n.Addr, validation.DeclRange, diags.Err().Error())
		}
		if !result.IsKnown() {
			log.Printf("[TRACE] evalVariableValidations: %s rule %s condition value is unknown, so skipping validation for now", n.Addr, validation.DeclRange)
			continue // We'll wait until we've learned more, then.
		}
		if result.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      "Validation condition expression must return either true or false, not null.",
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}
		var err error
		result, err = convert.Convert(result, cty.Bool)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     errInvalidCondition,
				Detail:      fmt.Sprintf("Invalid validation condition result value: %s.", tfdiags.FormatError(err)),
				Subject:     validation.Condition.Range().Ptr(),
				Expression:  validation.Condition,
				EvalContext: hclCtx,
			})
			continue
		}

		if result.False() {
			// Since we don't have a source expression for a root module
			// variable, we'll just report the error from the perspective
			// of the variable declaration itself.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  errInvalidValue,
				Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", validation.ErrorMessage, validation.DeclRange.String()),
				Subject:  n.Config.DeclRange.Ptr(),
			})
		}
	}

	return diags
}
