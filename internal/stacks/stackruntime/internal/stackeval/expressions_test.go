// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig/typeexpr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestEvalComponentInputVariables(t *testing.T) {
	decls := map[string]*configs.Variable{
		"plain":     {},
		"ephemeral": {Ephemeral: true},
		"sensitive": {Sensitive: true},
		"both":      {Ephemeral: true, Sensitive: true},
	}
	inputTy := cty.Object(map[string]cty.Type{
		"plain": cty.String, "ephemeral": cty.String, "sensitive": cty.String, "both": cty.String,
	})
	optionalInputTy := cty.ObjectWithOptionalAttrs(inputTy.AttributeTypes(), []string{"plain", "ephemeral", "sensitive", "both"})
	defaults := &typeexpr.Defaults{
		Type: optionalInputTy,
		DefaultValues: map[string]cty.Value{
			"plain":     cty.StringVal("default"),
			"ephemeral": cty.StringVal("default"),
			"sensitive": cty.StringVal("default"),
			"both":      cty.StringVal("default"),
		},
	}

	tests := map[string]struct {
		decls     map[string]*configs.Variable
		wantTy    cty.Type
		defaults  *typeexpr.Defaults
		inputs    hcl.Expression
		want      cty.Value
		wantDiags tfdiags.Diagnostics
	}{
		"known": {
			decls:  decls,
			wantTy: inputTy,
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("value"),
				"ephemeral": cty.StringVal("value"),
				"sensitive": cty.StringVal("value"),
				"both":      cty.StringVal("value"),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("value"),
				"ephemeral": cty.StringVal("value").Mark(marks.Ephemeral),
				"sensitive": cty.StringVal("value").Mark(marks.Sensitive),
				"both":      cty.StringVal("value").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"null": {
			decls:  decls,
			wantTy: inputTy,
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.NullVal(cty.String),
				"ephemeral": cty.NullVal(cty.String),
				"sensitive": cty.NullVal(cty.String),
				"both":      cty.NullVal(cty.String),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.NullVal(cty.String),
				"ephemeral": cty.NullVal(cty.String).Mark(marks.Ephemeral),
				"sensitive": cty.NullVal(cty.String).Mark(marks.Sensitive),
				"both":      cty.NullVal(cty.String).Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"unknown": {
			decls:  decls,
			wantTy: inputTy,
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.UnknownVal(cty.String),
				"ephemeral": cty.UnknownVal(cty.String),
				"sensitive": cty.UnknownVal(cty.String),
				"both":      cty.UnknownVal(cty.String),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.UnknownVal(cty.String),
				"ephemeral": cty.UnknownVal(cty.String).Mark(marks.Ephemeral),
				"sensitive": cty.UnknownVal(cty.String).Mark(marks.Sensitive),
				"both":      cty.UnknownVal(cty.String).Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"already sensitive": {
			decls:  decls,
			wantTy: inputTy,
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("value").Mark(marks.Sensitive),
				"ephemeral": cty.StringVal("value").Mark(marks.Sensitive),
				"sensitive": cty.StringVal("value").Mark(marks.Sensitive),
				"both":      cty.StringVal("value").Mark(marks.Sensitive),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("value").Mark(marks.Sensitive),
				"ephemeral": cty.StringVal("value").Mark(marks.Sensitive).Mark(marks.Ephemeral),
				"sensitive": cty.StringVal("value").Mark(marks.Sensitive),
				"both":      cty.StringVal("value").Mark(marks.Sensitive).Mark(marks.Ephemeral),
			}),
		},
		"default omitted": {
			decls:    decls,
			wantTy:   optionalInputTy,
			defaults: defaults,
			inputs:   hcltest.MockExprLiteral(cty.EmptyObjectVal),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("default"),
				"ephemeral": cty.StringVal("default").Mark(marks.Ephemeral),
				"sensitive": cty.StringVal("default").Mark(marks.Sensitive),
				"both":      cty.StringVal("default").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"default null": {
			decls:    decls,
			wantTy:   optionalInputTy,
			defaults: defaults,
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.NullVal(cty.String),
				"ephemeral": cty.NullVal(cty.String),
				"sensitive": cty.NullVal(cty.String),
				"both":      cty.NullVal(cty.String),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("default"),
				"ephemeral": cty.StringVal("default").Mark(marks.Ephemeral),
				"sensitive": cty.StringVal("default").Mark(marks.Sensitive),
				"both":      cty.StringVal("default").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"default without inputs": {
			decls:    decls,
			wantTy:   optionalInputTy,
			defaults: defaults,
			want: cty.ObjectVal(map[string]cty.Value{
				"plain":     cty.StringVal("default"),
				"ephemeral": cty.StringVal("default").Mark(marks.Ephemeral),
				"sensitive": cty.StringVal("default").Mark(marks.Sensitive),
				"both":      cty.StringVal("default").Mark(marks.Ephemeral).Mark(marks.Sensitive),
			}),
		},
		"preserve nested marks": {
			decls: map[string]*configs.Variable{"input": {Ephemeral: true}},
			wantTy: cty.Object(map[string]cty.Type{
				"input": cty.Object(map[string]cty.Type{"secret": cty.String, "plain": cty.String}),
			}),
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"input": cty.ObjectVal(map[string]cty.Value{
					"secret": cty.StringVal("value").Mark(marks.Sensitive).Mark(marks.Ephemeral),
					"plain":  cty.StringVal("plain"),
				}),
			})),
			want: cty.ObjectVal(map[string]cty.Value{
				"input": cty.ObjectVal(map[string]cty.Value{
					"secret": cty.StringVal("value").Mark(marks.Sensitive).Mark(marks.Ephemeral),
					"plain":  cty.StringVal("plain"),
				}).Mark(marks.Ephemeral),
			}),
		},
		"reject whole value": {
			decls: map[string]*configs.Variable{
				"input": {}, "other": {Ephemeral: true, Sensitive: true},
			},
			wantTy: cty.Object(map[string]cty.Type{"input": cty.String, "other": cty.String}),
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"input": cty.StringVal("secret").Mark(marks.Ephemeral),
				"other": cty.StringVal("other"),
			})),
			want: cty.UnknownVal(cty.Object(map[string]cty.Type{"input": cty.String, "other": cty.String})),
			wantDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Ephemeral value not allowed",
				Detail:   `The input variable "input" does not accept ephemeral values.`,
				Subject:  &hcl.Range{Filename: "MockExprLiteral"},
				Extra:    diagnosticCausedByEphemeral(true),
			}),
		},
		"reject nested value": {
			decls: map[string]*configs.Variable{
				"input": {}, "other": {Ephemeral: true, Sensitive: true},
			},
			wantTy: cty.Object(map[string]cty.Type{
				"input": cty.Object(map[string]cty.Type{"secret": cty.String}), "other": cty.String,
			}),
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"input": cty.ObjectVal(map[string]cty.Value{"secret": cty.StringVal("secret").Mark(marks.Ephemeral)}),
				"other": cty.StringVal("other"),
			})),
			want: cty.UnknownVal(cty.Object(map[string]cty.Type{
				"input": cty.Object(map[string]cty.Type{"secret": cty.String}), "other": cty.String,
			})),
			wantDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Ephemeral value not allowed",
				Detail:   `The input variable "input" does not accept ephemeral values, so the value for .secret is not compatible.`,
				Subject:  &hcl.Range{Filename: "MockExprLiteral"},
				Extra:    diagnosticCausedByEphemeral(true),
			}),
		},
		"reject unknown value": {
			decls: map[string]*configs.Variable{
				"input": {}, "other": {Ephemeral: true, Sensitive: true},
			},
			wantTy: cty.Object(map[string]cty.Type{"input": cty.String, "other": cty.String}),
			inputs: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
				"input": cty.UnknownVal(cty.String).Mark(marks.Ephemeral),
				"other": cty.StringVal("other"),
			})),
			want: cty.UnknownVal(cty.Object(map[string]cty.Type{"input": cty.String, "other": cty.String})),
			wantDiags: tfdiags.Diagnostics{}.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Ephemeral value not allowed",
				Detail:   `The input variable "input" does not accept ephemeral values.`,
				Subject:  &hcl.Range{Filename: "MockExprLiteral"},
				Extra:    diagnosticCausedByEphemeral(true),
			}),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := EvalComponentInputVariables(
				context.Background(), test.decls, test.wantTy, test.defaults, test.inputs, tfdiags.SourceRange{}, PlanPhase, newStaticExpressionScope(),
			)
			tfdiags.AssertDiagnosticsAndExtrasMatch(t, diags, test.wantDiags)
			if diff := cmp.Diff(test.want, got, ctydebug.CmpOptions); diff != "" {
				t.Errorf("wrong result (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvalExpr(t *testing.T) {
	t.Run("literal", func(t *testing.T) {
		ctx := context.Background()

		v := cty.StringVal("hello")
		expr := hcltest.MockExprLiteral(v)
		scope := newStaticExpressionScope()
		got, diags := EvalExpr(ctx, expr, PlanPhase, scope)
		if diags.HasErrors() {
			t.Errorf("unexpected diagnostics\n%s", diags.Err().Error())
		}
		if got, want := got, v; !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("valid reference", func(t *testing.T) {
		ctx := context.Background()

		v := cty.StringVal("indirect hello")
		expr := hcltest.MockExprTraversalSrc("local.example")
		scope := newStaticExpressionScope()
		scope.AddVal(stackaddrs.LocalValue{Name: "example"}, v)
		got, diags := EvalExpr(ctx, expr, PlanPhase, scope)
		if diags.HasErrors() {
			t.Errorf("unexpected diagnostics\n%s", diags.Err().Error())
		}
		if got, want := got, v; !want.RawEquals(got) {
			t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("invalid reference", func(t *testing.T) {
		ctx := context.Background()

		expr := hcltest.MockExprTraversalSrc("local.nonexist")
		scope := newStaticExpressionScope()
		_, diags := EvalExpr(ctx, expr, PlanPhase, scope)
		if !diags.HasErrors() {
			t.Errorf("unexpected success; want an error about local.nonexist not being defined")
		}
	})
	t.Run("multiple valid references", func(t *testing.T) {
		ctx := context.Background()

		// The following is aiming for coverage of all of the valid
		// stackaddrs.Referenceable implementations, since there's some
		// address-type-specific logic in EvalExpr. This also includes
		// some examples with extra traversal steps after the main address,
		// which tests that we can handle references where only a prefix
		// of the traversal is a referenceable object.
		expr := hcltest.MockExprList([]hcl.Expression{
			hcltest.MockExprTraversalSrc("local.example"),
			hcltest.MockExprTraversalSrc("var.example"),
			hcltest.MockExprTraversalSrc("component.example"),
			hcltest.MockExprTraversalSrc(`component.multi["foo"]`),
			hcltest.MockExprTraversalSrc("stack.example"),
			hcltest.MockExprTraversalSrc(`stack.multi["bar"]`),
			hcltest.MockExprTraversalSrc("provider.beep.boop"),
			hcltest.MockExprTraversalSrc(`provider.beep.boops["baz"]`),
			hcltest.MockExprTraversalSrc(`terraform.applying`),
		})

		scope := newStaticExpressionScope()
		scope.AddVal(stackaddrs.LocalValue{Name: "example"}, cty.StringVal("local value"))
		scope.AddVal(stackaddrs.InputVariable{Name: "example"}, cty.StringVal("input variable"))
		scope.AddVal(stackaddrs.Component{Name: "example"}, cty.StringVal("component singleton"))
		scope.AddVal(stackaddrs.Component{Name: "multi"}, cty.ObjectVal(map[string]cty.Value{
			"foo": cty.StringVal("component from for_each"),
		}))
		scope.AddVal(stackaddrs.StackCall{Name: "example"}, cty.StringVal("stack call singleton"))
		scope.AddVal(stackaddrs.StackCall{Name: "multi"}, cty.ObjectVal(map[string]cty.Value{
			"bar": cty.StringVal("stack call from for_each"),
		}))
		scope.AddVal(stackaddrs.ProviderConfigRef{ProviderLocalName: "beep", Name: "boop"}, cty.StringVal("provider config singleton"))
		scope.AddVal(stackaddrs.ProviderConfigRef{ProviderLocalName: "beep", Name: "boops"}, cty.ObjectVal(map[string]cty.Value{
			"baz": cty.StringVal("provider config from for_each"),
		}))
		scope.AddVal(stackaddrs.TerraformApplying, cty.StringVal("terraform.applying value")) // NOTE: Not a realistic terraform.applying value; just a placeholder to help exercise EvalExpr

		got, diags := EvalExpr(ctx, expr, PlanPhase, scope)
		if diags.HasErrors() {
			t.Errorf("unexpected diagnostics\n%s", diags.Err().Error())
		}
		want := cty.ListVal([]cty.Value{
			cty.StringVal("local value"),
			cty.StringVal("input variable"),
			cty.StringVal("component singleton"),
			cty.StringVal("component from for_each"),
			cty.StringVal("stack call singleton"),
			cty.StringVal("stack call from for_each"),
			cty.StringVal("provider config singleton"),
			cty.StringVal("provider config from for_each"),
			cty.StringVal("terraform.applying value"),
		})
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestReferencesInExpr(t *testing.T) {
	tests := []struct {
		exprSrc     string
		wantTargets []stackaddrs.Referenceable
	}{
		{
			`"hello"`,
			[]stackaddrs.Referenceable{},
		},
		{
			`var.foo`,
			[]stackaddrs.Referenceable{
				stackaddrs.InputVariable{
					Name: "foo",
				},
			},
		},
		{
			`var.foo + var.foo`,
			[]stackaddrs.Referenceable{
				stackaddrs.InputVariable{
					Name: "foo",
				},
				stackaddrs.InputVariable{
					Name: "foo",
				},
			},
		},
		{
			`local.bar`,
			[]stackaddrs.Referenceable{
				stackaddrs.LocalValue{
					Name: "bar",
				},
			},
		},
		{
			`component.foo["bar"]`,
			[]stackaddrs.Referenceable{
				stackaddrs.Component{
					Name: "foo",
				},
			},
		},
		{
			`stack.foo["bar"]`,
			[]stackaddrs.Referenceable{
				stackaddrs.StackCall{
					Name: "foo",
				},
			},
		},
		{
			`provider.foo.bar["baz"]`,
			[]stackaddrs.Referenceable{
				stackaddrs.ProviderConfigRef{
					ProviderLocalName: "foo",
					Name:              "bar",
				},
			},
		},
		{
			`terraform.applying`,
			[]stackaddrs.Referenceable{
				stackaddrs.TerraformApplying,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.exprSrc, func(t *testing.T) {
			var diags tfdiags.Diagnostics
			expr, hclDiags := hclsyntax.ParseExpression([]byte(test.exprSrc), "", hcl.InitialPos)
			diags = diags.Append(hclDiags)
			assertNoDiagnostics(t, diags)

			gotRefs := ReferencesInExpr(expr)
			gotTargets := make([]stackaddrs.Referenceable, len(gotRefs))
			for i, ref := range gotRefs {
				gotTargets[i] = ref.Target
			}

			if diff := cmp.Diff(test.wantTargets, gotTargets); diff != "" {
				t.Errorf("wrong reference targets\n%s", diff)
			}
		})
	}
}

func TestEvalBody(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()

		body := hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"literal": {
					Name: "literal",
					Expr: hcltest.MockExprLiteral(cty.StringVal("literal value")),
				},
				"reference": {
					Name: "reference",
					Expr: hcltest.MockExprTraversalSrc("local.example"),
				},
			},
		})

		scope := newStaticExpressionScope()
		scope.AddVal(stackaddrs.LocalValue{Name: "example"}, cty.StringVal("reference value"))

		spec := hcldec.ObjectSpec{
			"lit": &hcldec.AttrSpec{
				Name: "literal",
				Type: cty.String,
			},
			"ref": &hcldec.AttrSpec{
				Name: "reference",
				Type: cty.String,
			},
		}

		got, diags := EvalBody(ctx, body, spec, PlanPhase, scope)
		if diags.HasErrors() {
			t.Errorf("unexpected diagnostics\n%s", diags.Err().Error())
		}
		want := cty.ObjectVal(map[string]cty.Value{
			"lit": cty.StringVal("literal value"),
			"ref": cty.StringVal("reference value"),
		})
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}

func TestPerEvalPhase(t *testing.T) {
	pep := perEvalPhase[string]{}

	forPlan := pep.For(PlanPhase)
	if forPlan == nil || *forPlan != "" {
		t.Error("value should initially be the zero value of T")
	}
	forApply := pep.For(ApplyPhase)
	if forApply == nil || *forApply != "" {
		t.Error("value should initially be the zero value of T")
	}

	*forPlan = "plan phase"
	*forApply = "apply phase"

	forPlan = pep.For(PlanPhase)
	if forPlan == nil || *forPlan != "plan phase" {
		t.Error("didn't remember the value for the plan phase")
	}

	forApply = pep.For(ApplyPhase)
	if forApply == nil || *forApply != "apply phase" {
		t.Error("didn't remember the value for the apply phase")
	}

	*(pep.For(ValidatePhase)) = "validate phase"

	gotVals := map[EvalPhase]string{}
	pep.Each(func(ep EvalPhase, t *string) {
		gotVals[ep] = *t
	})
	wantVals := map[EvalPhase]string{
		ValidatePhase: "validate phase",
		PlanPhase:     "plan phase",
		ApplyPhase:    "apply phase",
	}
	if diff := cmp.Diff(wantVals, gotVals); diff != "" {
		t.Errorf("wrong values\n%s", diff)
	}
}

// staticReferenceable is an implementation of [Referenceable] that just
// returns a statically-provided value, as an aid to unit testing.
type staticReferenceable struct {
	v cty.Value
}

var _ Referenceable = staticReferenceable{}

// ExprReferenceValue implements Referenceable.
func (r staticReferenceable) ExprReferenceValue(context.Context, EvalPhase) cty.Value {
	return r.v
}

// staticExpressionScope is an implementation of [ExpressionScope] that
// has a static table of referenceable objects that it returns on request,
// as an aid to unit testing.
type staticExpressionScope struct {
	vs collections.Map[stackaddrs.Referenceable, Referenceable]
}

var _ ExpressionScope = staticExpressionScope{}

func newStaticExpressionScope() staticExpressionScope {
	return staticExpressionScope{
		vs: collections.NewMapFunc[stackaddrs.Referenceable, Referenceable](
			func(r stackaddrs.Referenceable) collections.UniqueKey[stackaddrs.Referenceable] {
				// Since this is just for testing purposes we'll use just
				// string comparison for our key lookups. This should be fine
				// as long as we continue to preserve the property that there
				// is no overlap between string representations of different
				// refereceable types, which is true at the time of writing
				// this function.
				return staticExpressionScopeKey(r.String())
			},
		),
	}
}

// ResolveExpressionReference implements ExpressionScope.
func (s staticExpressionScope) ResolveExpressionReference(_ context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret, ok := s.vs.GetOk(ref.Target)
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid reference",
			Detail:   fmt.Sprintf("The address %s does not match anything known to this test-focused static expression scope.", ref.Target.String()),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
		return nil, diags
	}
	return ret, diags
}

// ExternalFunctions implements ExpressionScope
func (s staticExpressionScope) ExternalFunctions(_ context.Context) (lang.ExternalFuncs, tfdiags.Diagnostics) {
	return lang.ExternalFuncs{}, nil
}

// PlanTimestamp implements ExpressionScope
func (s staticExpressionScope) PlanTimestamp() time.Time {
	return time.Now().UTC()
}

// Add makes the given object available in the scope at the given address.
func (s staticExpressionScope) Add(addr stackaddrs.Referenceable, obj Referenceable) {
	s.vs.Put(addr, obj)
}

// AddVal is an convenience wrapper for Add which wraps the given value in a
// [staticReferenceable] before adding it.
func (s staticExpressionScope) AddVal(addr stackaddrs.Referenceable, val cty.Value) {
	s.Add(addr, staticReferenceable{val})
}

type staticExpressionScopeKey string

// IsUniqueKey implements collections.UniqueKey.
func (staticExpressionScopeKey) IsUniqueKey(stackaddrs.Referenceable) {}
