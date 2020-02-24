// Package tryfunc contains some optional functions that can be exposed in
// HCL-based languages to allow authors to test whether a particular expression
// can succeed and take dynamic action based on that result.
//
// These functions are implemented in terms of the customdecode extension from
// the sibling directory "customdecode", and so they are only useful when
// used within an HCL EvalContext. Other systems using cty functions are
// unlikely to support the HCL-specific "customdecode" extension.
package tryfunc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/customdecode"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// TryFunc is a variadic function that tries to evaluate all of is arguments
// in sequence until one succeeds, in which case it returns that result, or
// returns an error if none of them succeed.
var TryFunc function.Function

// CanFunc tries to evaluate the expression given in its first argument.
var CanFunc function.Function

func init() {
	TryFunc = function.New(&function.Spec{
		VarParam: &function.Parameter{
			Name: "expressions",
			Type: customdecode.ExpressionClosureType,
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			v, err := try(args)
			if err != nil {
				return cty.NilType, err
			}
			return v.Type(), nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return try(args)
		},
	})
	CanFunc = function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "expression",
				Type: customdecode.ExpressionClosureType,
			},
		},
		Type: function.StaticReturnType(cty.Bool),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return can(args[0])
		},
	})
}

func try(args []cty.Value) (cty.Value, error) {
	if len(args) == 0 {
		return cty.NilVal, errors.New("at least one argument is required")
	}

	// We'll collect up all of the diagnostics we encounter along the way
	// and report them all if none of the expressions succeed, so that the
	// user might get some hints on how to make at least one succeed.
	var diags hcl.Diagnostics
	for _, arg := range args {
		closure := customdecode.ExpressionClosureFromVal(arg)
		if dependsOnUnknowns(closure.Expression, closure.EvalContext) {
			// We can't safely decide if this expression will succeed yet,
			// and so our entire result must be unknown until we have
			// more information.
			return cty.DynamicVal, nil
		}

		v, moreDiags := closure.Value()
		diags = append(diags, moreDiags...)
		if moreDiags.HasErrors() {
			continue // try the next one, if there is one to try
		}
		return v, nil // ignore any accumulated diagnostics if one succeeds
	}

	// If we fall out here then none of the expressions succeeded, and so
	// we must have at least one diagnostic and we'll return all of them
	// so that the user can see the errors related to whichever one they
	// were expecting to have succeeded in this case.
	//
	// Because our function must return a single error value rather than
	// diagnostics, we'll construct a suitable error message string
	// that will make sense in the context of the function call failure
	// diagnostic HCL will eventually wrap this in.
	var buf strings.Builder
	buf.WriteString("no expression succeeded:\n")
	for _, diag := range diags {
		if diag.Subject != nil {
			buf.WriteString(fmt.Sprintf("- %s (at %s)\n  %s\n", diag.Summary, diag.Subject, diag.Detail))
		} else {
			buf.WriteString(fmt.Sprintf("- %s\n  %s\n", diag.Summary, diag.Detail))
		}
	}
	buf.WriteString("\nAt least one expression must produce a successful result")
	return cty.NilVal, errors.New(buf.String())
}

func can(arg cty.Value) (cty.Value, error) {
	closure := customdecode.ExpressionClosureFromVal(arg)
	if dependsOnUnknowns(closure.Expression, closure.EvalContext) {
		// Can't decide yet, then.
		return cty.UnknownVal(cty.Bool), nil
	}

	_, diags := closure.Value()
	if diags.HasErrors() {
		return cty.False, nil
	}
	return cty.True, nil
}

// dependsOnUnknowns returns true if any of the variables that the given
// expression might access are unknown values or contain unknown values.
//
// This is a conservative result that prefers to return true if there's any
// chance that the expression might derive from an unknown value during its
// evaluation; it is likely to produce false-positives for more complex
// expressions involving deep data structures.
func dependsOnUnknowns(expr hcl.Expression, ctx *hcl.EvalContext) bool {
	for _, traversal := range expr.Variables() {
		val, diags := traversal.TraverseAbs(ctx)
		if diags.HasErrors() {
			// If the traversal returned a definitive error then it must
			// not traverse through any unknowns.
			continue
		}
		if !val.IsWhollyKnown() {
			// The value will be unknown if either it refers directly to
			// an unknown value or if the traversal moves through an unknown
			// collection. We're using IsWhollyKnown, so this also catches
			// situations where the traversal refers to a compound data
			// structure that contains any unknown values. That's important,
			// because during evaluation the expression might evaluate more
			// deeply into this structure and encounter the unknowns.
			return true
		}
	}
	return false
}
