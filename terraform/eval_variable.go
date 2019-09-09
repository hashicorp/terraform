package terraform

import (
	"fmt"
	"log"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// EvalSetModuleCallArguments is an EvalNode implementation that sets values
// for arguments of a child module call, for later retrieval during
// expression evaluation.
type EvalSetModuleCallArguments struct {
	Module addrs.ModuleCallInstance
	Values map[string]cty.Value
}

// TODO: test
func (n *EvalSetModuleCallArguments) Eval(ctx EvalContext) (interface{}, error) {
	ctx.SetModuleCallArguments(n.Module, n.Values)
	return nil, nil
}

// EvalModuleCallArgument is an EvalNode implementation that produces the value
// for a particular variable as will be used by a child module instance.
//
// The result is written into the map given in Values, with its key
// set to the local name of the variable, disregarding the module instance
// address. Any existing values in that map are deleted first. This weird
// interface is a result of trying to be convenient for use with
// EvalContext.SetModuleCallArguments, which expects a map to merge in with
// any existing arguments.
type EvalModuleCallArgument struct {
	Addr   addrs.InputVariable
	Config *configs.Variable
	Expr   hcl.Expression

	// If this flag is set, any diagnostics are discarded and this operation
	// will always succeed, though may produce an unknown value in the
	// event of an error.
	IgnoreDiagnostics bool

	Values map[string]cty.Value
}

func (n *EvalModuleCallArgument) Eval(ctx EvalContext) (interface{}, error) {
	// Clear out the existing mapping
	for k := range n.Values {
		delete(n.Values, k)
	}

	wantType := n.Config.Type
	name := n.Addr.Name
	expr := n.Expr

	if expr == nil {
		// Should never happen, but we'll bail out early here rather than
		// crash in case it does. We set no value at all in this case,
		// making a subsequent call to EvalContext.SetModuleCallArguments
		// a no-op.
		log.Printf("[ERROR] attempt to evaluate %s with nil expression", n.Addr.String())
		return nil, nil
	}

	val, diags := ctx.EvaluateExpr(expr, cty.DynamicPseudoType, nil)

	// We intentionally passed DynamicPseudoType to EvaluateExpr above because
	// now we can do our own local type conversion and produce an error message
	// with better context if it fails.
	var convErr error
	val, convErr = convert.Convert(val, wantType)
	if convErr != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid value for module argument",
			Detail: fmt.Sprintf(
				"The given value is not suitable for child module variable %q defined at %s: %s.",
				name, n.Config.DeclRange.String(), convErr,
			),
			Subject: expr.Range().Ptr(),
		})
		// We'll return a placeholder unknown value to avoid producing
		// redundant downstream errors.
		val = cty.UnknownVal(wantType)
	}

	n.Values[name] = val
	if n.IgnoreDiagnostics {
		return nil, nil
	}
	return nil, diags.ErrWithWarnings()
}

// hclTypeName returns the name of the type that would represent this value in
// a config file, or falls back to the Go type name if there's no corresponding
// HCL type. This is used for formatted output, not for comparing types.
func hclTypeName(i interface{}) string {
	switch k := reflect.Indirect(reflect.ValueOf(i)).Kind(); k {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Array, reflect.Slice:
		return "list"
	case reflect.Map:
		return "map"
	case reflect.String:
		return "string"
	default:
		// fall back to the Go type if there's no match
		return k.String()
	}
}
