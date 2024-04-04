// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/namedvals"
)

func TestNodeRootVariableExecute(t *testing.T) {
	t.Run("type conversion", func(t *testing.T) {
		ctx := new(MockEvalContext)

		n := &NodeRootVariable{
			Addr: addrs.InputVariable{Name: "foo"},
			Config: &configs.Variable{
				Name:           "foo",
				Type:           cty.String,
				ConstraintType: cty.String,
			},
			RawValue: &InputValue{
				Value:      cty.True,
				SourceType: ValueFromUnknown,
			},
		}

		ctx.NamedValuesState = namedvals.NewState()

		diags := n.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error: %s", diags.Err())
		}

		absAddr := addrs.RootModuleInstance.InputVariable(n.Addr.Name)
		if !ctx.NamedValues().HasInputVariableValue(absAddr) {
			t.Fatalf("no result was registered")
		}
		if got, want := ctx.NamedValues().GetInputVariableValue(absAddr), cty.StringVal("true"); !want.RawEquals(got) {
			// NOTE: The given value was cty.Bool but the type constraint was
			// cty.String, so it was NodeRootVariable's responsibility to convert
			// as part of preparing the "final value".
			t.Errorf("wrong value for ctx.SetRootModuleArgument\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("validation", func(t *testing.T) {
		ctx := new(MockEvalContext)

		// Validation is actually handled by a separate node of type
		// nodeVariableValidation, so this test will combine NodeRootVariable
		// and nodeVariableValidation to check that they work together
		// correctly in integration.

		ctx.NamedValuesState = namedvals.NewState()

		// The variable validation function gets called with Terraform's
		// built-in functions available, so we need a minimal scope just for
		// it to get the functions from.
		ctx.EvaluationScopeScope = &lang.Scope{}

		n := &NodeRootVariable{
			Addr: addrs.InputVariable{Name: "foo"},
			Config: &configs.Variable{
				Name:           "foo",
				Type:           cty.Number,
				ConstraintType: cty.Number,
				Validations: []*configs.CheckRule{
					{
						Condition: fakeHCLExpressionFunc(func(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
							// This returns true only if the given variable value
							// is exactly cty.Number, which allows us to verify
							// that we were given the value _after_ type
							// conversion.
							// This had previously not been handled correctly,
							// as reported in:
							//     https://github.com/hashicorp/terraform/issues/29899
							vars := ctx.Variables["var"]
							if vars == cty.NilVal || !vars.Type().IsObjectType() || !vars.Type().HasAttribute("foo") {
								t.Logf("var.foo isn't available")
								return cty.False, nil
							}
							val := vars.GetAttr("foo")
							if val == cty.NilVal || val.Type() != cty.Number {
								t.Logf("var.foo is %#v; want a number", val)
								return cty.False, nil
							}
							return cty.True, nil
						}),
						ErrorMessage: hcltest.MockExprLiteral(cty.StringVal("Must be a number.")),
					},
				},
			},
			RawValue: &InputValue{
				// Note: This is a string, but the variable's type constraint
				// is number so it should be converted before use.
				Value:      cty.StringVal("5"),
				SourceType: ValueFromUnknown,
			},
			Planning: true,
		}
		configAddr, validationRules, defnRange := n.variableValidationRules()
		validateN := &nodeVariableValidation{
			configAddr: configAddr,
			rules:      validationRules,
			defnRange:  defnRange,
		}

		ctx.ChecksState = checks.NewState(&configs.Config{
			Module: &configs.Module{
				Variables: map[string]*configs.Variable{
					"foo": n.Config,
				},
			},
		})

		diags := n.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error from NodeRootVariable: %s", diags.Err())
		}
		diags = validateN.Execute(ctx, walkApply)
		if diags.HasErrors() {
			t.Fatalf("unexpected error from nodeVariableValidation: %s", diags.Err())
		}

		absAddr := addrs.RootModuleInstance.InputVariable(n.Addr.Name)
		if !ctx.NamedValues().HasInputVariableValue(absAddr) {
			t.Fatalf("no result value for input variable")
		}
		if got, want := ctx.NamedValues().GetInputVariableValue(absAddr), cty.NumberIntVal(5); !want.RawEquals(got) {
			// NOTE: The given value was cty.Bool but the type constraint was
			// cty.String, so it was NodeRootVariable's responsibility to convert
			// as part of preparing the "final value".
			t.Errorf("wrong value for ctx.SetRootModuleArgument\ngot:  %#v\nwant: %#v", got, want)
		}
		if status := ctx.Checks().ObjectCheckStatus(n.Addr.Absolute(addrs.RootModuleInstance)); status != checks.StatusPass {
			t.Errorf("expected checks to pass but go %s instead", status)
		}
	})
}

// fakeHCLExpressionFunc is a fake implementation of hcl.Expression that just
// directly produces a value with direct Go code.
//
// An expression of this type has no references and so it cannot access any
// variables from the EvalContext unless something else arranges for them
// to be guaranteed available. For example, custom variable validations just
// unconditionally have access to the variable they are validating regardless
// of references.
type fakeHCLExpressionFunc func(*hcl.EvalContext) (cty.Value, hcl.Diagnostics)

var _ hcl.Expression = fakeHCLExpressionFunc(nil)

func (f fakeHCLExpressionFunc) Value(ctx *hcl.EvalContext) (cty.Value, hcl.Diagnostics) {
	return f(ctx)
}

func (f fakeHCLExpressionFunc) Variables() []hcl.Traversal {
	return nil
}

func (f fakeHCLExpressionFunc) Range() hcl.Range {
	return hcl.Range{
		Filename: "fake",
		Start:    hcl.InitialPos,
		End:      hcl.InitialPos,
	}
}

func (f fakeHCLExpressionFunc) StartRange() hcl.Range {
	return f.Range()
}
