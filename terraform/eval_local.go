package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/zclconf/go-cty/cty"
)

// EvalLocal is an EvalNode implementation that evaluates the
// expression for a local value and writes it into a transient part of
// the state.
type EvalLocal struct {
	Addr addrs.LocalValue
	Expr hcl.Expression
}

func (n *EvalLocal) Eval(ctx EvalContext) (interface{}, error) {
	val, diags := ctx.EvaluateExpr(n.Expr, cty.DynamicPseudoType, nil)
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	state, lock := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write local value to nil state")
	}

	// Get a write lock so we can access the state
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		mod = state.AddModule(ctx.Path())
	}

	// Lower the value to the legacy form that our state structures still expect.
	// FIXME: Update mod.Locals to be a map[string]cty.Value .
	legacyVal := hcl2shim.ConfigValueFromHCL2(val)

	if mod.Locals == nil {
		// initialize
		mod.Locals = map[string]interface{}{}
	}
	mod.Locals[n.Addr.Name] = legacyVal

	return nil, nil
}

// EvalDeleteLocal is an EvalNode implementation that deletes a Local value
// from the state. Locals aren't persisted, but we don't need to evaluate them
// during destroy.
type EvalDeleteLocal struct {
	Addr addrs.LocalValue
}

func (n *EvalDeleteLocal) Eval(ctx EvalContext) (interface{}, error) {
	state, lock := ctx.State()
	if state == nil {
		return nil, nil
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		return nil, nil
	}

	delete(mod.Locals, n.Addr.Name)

	return nil, nil
}
