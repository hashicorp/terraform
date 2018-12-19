package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalLocal is an EvalNode implementation that evaluates the
// expression for a local value and writes it into a transient part of
// the state.
type EvalLocal struct {
	Addr addrs.LocalValue
	Expr hcl.Expression
}

func (n *EvalLocal) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	// We ignore diags here because any problems we might find will be found
	// again in EvaluateExpr below.
	refs, _ := lang.ReferencesInExpr(n.Expr)
	for _, ref := range refs {
		if ref.Subject == n.Addr {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Self-referencing local value",
				Detail:   fmt.Sprintf("Local value %s cannot use its own result as part of its expression.", n.Addr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
				Context:  n.Expr.Range().Ptr(),
			})
		}
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	val, moreDiags := ctx.EvaluateExpr(n.Expr, cty.DynamicPseudoType, nil)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags.Err()
	}

	state := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write local value to nil state")
	}

	state.SetLocalValue(n.Addr.Absolute(ctx.Path()), val)

	return nil, nil
}

// EvalDeleteLocal is an EvalNode implementation that deletes a Local value
// from the state. Locals aren't persisted, but we don't need to evaluate them
// during destroy.
type EvalDeleteLocal struct {
	Addr addrs.LocalValue
}

func (n *EvalDeleteLocal) Eval(ctx EvalContext) (interface{}, error) {
	state := ctx.State()
	if state == nil {
		return nil, nil
	}

	state.RemoveLocalValue(n.Addr.Absolute(ctx.Path()))
	return nil, nil
}
