package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// EvalDeleteOutput is an EvalNode implementation that deletes an output
// from the state.
type EvalDeleteOutput struct {
	Addr addrs.OutputValue
}

// TODO: test
func (n *EvalDeleteOutput) Eval(ctx EvalContext) (interface{}, error) {
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

	delete(mod.Outputs, n.Addr.Name)

	return nil, nil
}

// EvalWriteOutput is an EvalNode implementation that writes the output
// for the given name to the current state.
type EvalWriteOutput struct {
	Addr      addrs.OutputValue
	Sensitive bool
	Expr      hcl.Expression
	// ContinueOnErr allows interpolation to fail during Input
	ContinueOnErr bool
}

// TODO: test
func (n *EvalWriteOutput) Eval(ctx EvalContext) (interface{}, error) {
	// This has to run before we have a state lock, since evaluation also
	// reads the state
	val, diags := ctx.EvaluateExpr(n.Expr, cty.DynamicPseudoType, nil)
	// We'll handle errors below, after we have loaded the module.

	state, lock := ctx.State()
	if state == nil {
		return nil, fmt.Errorf("cannot write state to nil state")
	}

	// Get a write lock so we can access this instance
	lock.Lock()
	defer lock.Unlock()
	// Look for the module state. If we don't have one, create it.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		mod = state.AddModule(ctx.Path())
	}

	// handling the interpolation error
	if diags.HasErrors() {
		if n.ContinueOnErr || flagWarnOutputErrors {
			log.Printf("[ERROR] Output interpolation %q failed: %s", n.Addr.Name, diags.Err())
			// if we're continuing, make sure the output is included, and
			// marked as unknown
			mod.Outputs[n.Addr.Name] = &OutputState{
				Type:  "string",
				Value: config.UnknownVariableValue,
			}
			return nil, EvalEarlyExitError{}
		}
		return nil, diags.Err()
	}

	ty := val.Type()
	switch {
	case ty.IsPrimitiveType():
		// For now we record all primitive types as strings, for compatibility
		// with our existing state formats.
		// FIXME: Revise the state format to support any type.
		var valueTyped string
		switch {
		case !val.IsKnown():
			// Legacy handling of unknown values as a special string.
			valueTyped = config.UnknownVariableValue
		case val.IsNull():
			// State doesn't currently support null, so we'll save as empty string.
			valueTyped = ""
		default:
			err := gocty.FromCtyValue(val, &valueTyped)
			if err != nil {
				// Should never happen, because all primitives can convert to string.
				return nil, fmt.Errorf("cannot marshal %#v for storage in state: %s", err)
			}
		}
		mod.Outputs[n.Addr.Name] = &OutputState{
			Type:      "string",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		// For now we'll use our legacy storage forms for list-like types.
		// This produces a []interface{}.
		valueTyped := hcl2shim.ConfigValueFromHCL2(val)
		mod.Outputs[n.Addr.Name] = &OutputState{
			Type:      "list",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	case ty.IsMapType() || ty.IsObjectType():
		// For now we'll use our legacy storage forms for map-like types.
		// This produces a map[string]interface{}.
		valueTyped := hcl2shim.ConfigValueFromHCL2(val)
		mod.Outputs[n.Addr.Name] = &OutputState{
			Type:      "map",
			Sensitive: n.Sensitive,
			Value:     valueTyped,
		}
	default:
		return nil, fmt.Errorf("output %s is not a valid type (%s)", n.Addr.Name, ty.FriendlyName())
	}

	return nil, nil
}
