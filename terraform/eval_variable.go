package terraform

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// EvalTypeCheckVariable is an EvalNode which ensures that the variable
// values which are assigned as inputs to a module (including the root)
// match the types which are either declared for the variables explicitly
// or inferred from the default values.
//
// In order to achieve this three things are required:
//     - a map of the proposed variable values
//     - the configuration tree of the module in which the variable is
//       declared
//     - the path to the module (so we know which part of the tree to
//       compare the values against).
type EvalTypeCheckVariable struct {
	Variables  map[string]interface{}
	ModulePath []string
	ModuleTree *module.Tree
}

func (n *EvalTypeCheckVariable) Eval(ctx EvalContext) (interface{}, error) {
	currentTree := n.ModuleTree
	for _, pathComponent := range n.ModulePath[1:] {
		currentTree = currentTree.Children()[pathComponent]
	}
	targetConfig := currentTree.Config()

	prototypes := make(map[string]config.VariableType)
	for _, variable := range targetConfig.Variables {
		prototypes[variable.Name] = variable.Type()
	}

	// Only display a module in an error message if we are not in the root module
	modulePathDescription := fmt.Sprintf(" in module %s", strings.Join(n.ModulePath[1:], "."))
	if len(n.ModulePath) == 1 {
		modulePathDescription = ""
	}

	for name, declaredType := range prototypes {
		proposedValue, ok := n.Variables[name]
		if !ok {
			// This means the default value should be used as no overriding value
			// has been set. Therefore we should continue as no check is necessary.
			continue
		}

		if proposedValue == config.UnknownVariableValue {
			continue
		}

		switch declaredType {
		case config.VariableTypeString:
			switch proposedValue.(type) {
			case string:
				continue
			default:
				return nil, fmt.Errorf("variable %s%s should be type %s, got %s",
					name, modulePathDescription, declaredType.Printable(), hclTypeName(proposedValue))
			}
		case config.VariableTypeMap:
			switch proposedValue.(type) {
			case map[string]interface{}:
				continue
			default:
				return nil, fmt.Errorf("variable %s%s should be type %s, got %s",
					name, modulePathDescription, declaredType.Printable(), hclTypeName(proposedValue))
			}
		case config.VariableTypeList:
			switch proposedValue.(type) {
			case []interface{}:
				continue
			default:
				return nil, fmt.Errorf("variable %s%s should be type %s, got %s",
					name, modulePathDescription, declaredType.Printable(), hclTypeName(proposedValue))
			}
		default:
			return nil, fmt.Errorf("variable %s%s should be type %s, got type string",
				name, modulePathDescription, declaredType.Printable())
		}
	}

	return nil, nil
}

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
