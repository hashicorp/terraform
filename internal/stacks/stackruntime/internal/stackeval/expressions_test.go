// Copyright (c) HashiCorp, Inc.
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
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

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

			gotRefs := ReferencesInExpr(context.Background(), expr)
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
func (r staticReferenceable) ExprReferenceValue(ctx context.Context, phase EvalPhase) cty.Value {
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
func (s staticExpressionScope) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
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
func (s staticExpressionScope) ExternalFunctions(ctx context.Context) (lang.ExternalFuncs, func(), tfdiags.Diagnostics) {
	return lang.ExternalFuncs{}, func() {}, nil
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
