package terraform

import (
	"fmt"
	"log"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/instances"
	"github.com/hashicorp/terraform/tfdiags"
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
	Addr           addrs.InputVariable
	Config         *configs.Variable
	Expr           hcl.Expression
	ModuleInstance addrs.ModuleInstance

	Values map[string]cty.Value

	// validateOnly indicates that this evaluation is only for config
	// validation, and we will not have any expansion module instance
	// repetition data.
	validateOnly bool
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

	var moduleInstanceRepetitionData instances.RepetitionData

	switch {
	case n.validateOnly:
		// the instance expander does not track unknown expansion values, so we
		// have to assume all RepetitionData is unknown.
		moduleInstanceRepetitionData = instances.RepetitionData{
			CountIndex: cty.UnknownVal(cty.Number),
			EachKey:    cty.UnknownVal(cty.String),
			EachValue:  cty.DynamicVal,
		}

	default:
		// Get the repetition data for this module instance,
		// so we can create the appropriate scope for evaluating our expression
		moduleInstanceRepetitionData = ctx.InstanceExpander().GetModuleInstanceRepetitionData(n.ModuleInstance)
	}

	scope := ctx.EvaluationScope(nil, moduleInstanceRepetitionData)
	val, diags := scope.EvalExpr(expr, cty.DynamicPseudoType)

	// We intentionally passed DynamicPseudoType to EvalExpr above because
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
	return nil, diags.ErrWithWarnings()
}

// evalVariableValidations is an EvalNode implementation that ensures that
// all of the configured custom validations for a variable are passing.
//
// This must be used only after any side-effects that make the value of the
// variable available for use in expression evaluation, such as
// EvalModuleCallArgument for variables in descendent modules.
type evalVariableValidations struct {
	Addr   addrs.AbsInputVariableInstance
	Config *configs.Variable

	// Expr is the expression that provided the value for the variable, if any.
	// This will be nil for root module variables, because their values come
	// from outside the configuration.
	Expr hcl.Expression
}

func (n *evalVariableValidations) Eval(ctx EvalContext) (interface{}, error) {
	if n.Config == nil || len(n.Config.Validations) == 0 {
		log.Printf("[TRACE] evalVariableValidations: not active for %s, so skipping", n.Addr)
		return nil, nil
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
	val := ctx.GetVariableValue(n.Addr)
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
			if n.Expr != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  errInvalidValue,
					Detail:   fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", validation.ErrorMessage, validation.DeclRange.String()),
					Subject:  n.Expr.Range().Ptr(),
				})
			} else {
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
