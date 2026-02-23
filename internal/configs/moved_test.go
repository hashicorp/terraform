// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestMovedBlock_decode(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
		End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
	}

	foo_expr := hcltest.MockExprTraversalSrc("test_instance.foo")
	bar_expr := hcltest.MockExprTraversalSrc("test_instance.bar")

	foo_index_expr := hcltest.MockExprTraversalSrc("test_instance.foo[1]")
	bar_index_expr := hcltest.MockExprTraversalSrc("test_instance.bar[\"one\"]")

	mod_foo_expr := hcltest.MockExprTraversalSrc("module.foo")
	mod_bar_expr := hcltest.MockExprTraversalSrc("module.bar")
	for_each_expr := hcltest.MockExprTraversalSrc("var.moves")

	tests := map[string]struct {
		input *hcl.Block
		want  *Moved
		err   string
	}{
		"success": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustMoveEndpointFromExpr(foo_expr),
				To:        mustMoveEndpointFromExpr(bar_expr),
				FromExpr:  foo_expr,
				ToExpr:    bar_expr,
				DeclRange: blockRange,
			},
			``,
		},
		"indexed resources": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_index_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_index_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustMoveEndpointFromExpr(foo_index_expr),
				To:        mustMoveEndpointFromExpr(bar_index_expr),
				FromExpr:  foo_index_expr,
				ToExpr:    bar_index_expr,
				DeclRange: blockRange,
			},
			``,
		},
		"modules": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: mod_foo_expr,
						},
						"to": {
							Name: "to",
							Expr: mod_bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustMoveEndpointFromExpr(mod_foo_expr),
				To:        mustMoveEndpointFromExpr(mod_bar_expr),
				FromExpr:  mod_foo_expr,
				ToExpr:    mod_bar_expr,
				DeclRange: blockRange,
			},
			``,
		},
		"success with for_each expression retained": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
						"for_each": {
							Name: "for_each",
							Expr: for_each_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustMoveEndpointFromExpr(foo_expr),
				To:        mustMoveEndpointFromExpr(bar_expr),
				FromExpr:  foo_expr,
				ToExpr:    bar_expr,
				ForEach:   for_each_expr,
				DeclRange: blockRange,
			},
			``,
		},
		"error: missing argument": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				From:      mustMoveEndpointFromExpr(foo_expr),
				FromExpr:  foo_expr,
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: type mismatch": {
			&hcl.Block{
				Type: "moved",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"to": {
							Name: "to",
							Expr: foo_expr,
						},
						"from": {
							Name: "from",
							Expr: mod_foo_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Moved{
				To:        mustMoveEndpointFromExpr(foo_expr),
				From:      mustMoveEndpointFromExpr(mod_foo_expr),
				FromExpr:  mod_foo_expr,
				ToExpr:    foo_expr,
				DeclRange: blockRange,
			},
			"Invalid \"moved\" addresses",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeMovedBlock(test.input)

			if diags.HasErrors() {
				if test.err == "" {
					t.Fatalf("unexpected error: %s", diags.Errs())
				}
				if gotErr := diags[0].Summary; gotErr != test.err {
					t.Errorf("wrong error, got %q, want %q", gotErr, test.err)
				}
			} else if test.err != "" {
				t.Fatal("expected error")
			}

			if diff := cmp.Diff(test.want.From, got.From, cmp.AllowUnexported(addrs.MoveEndpoint{})); diff != "" {
				t.Fatalf("wrong from endpoint\n%s", diff)
			}
			if diff := cmp.Diff(test.want.To, got.To, cmp.AllowUnexported(addrs.MoveEndpoint{})); diff != "" {
				t.Fatalf("wrong to endpoint\n%s", diff)
			}
			if got.DeclRange != test.want.DeclRange {
				t.Fatalf("wrong decl range\ngot:  %#v\nwant: %#v", got.DeclRange, test.want.DeclRange)
			}
			if !exprTraversalEqual(got.FromExpr, test.want.FromExpr) {
				t.Fatalf("wrong from expression")
			}
			if !exprTraversalEqual(got.ToExpr, test.want.ToExpr) {
				t.Fatalf("wrong to expression")
			}
			if !exprTraversalEqual(got.ForEach, test.want.ForEach) {
				t.Fatalf("wrong for_each expression")
			}
		})
	}
}

func TestMovedBlock_decodeForEachDynamicEndpoints(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
		End:      hcl.Pos{Line: 6, Column: 2, Byte: 90},
	}

	parseExpr := func(src string) hcl.Expression {
		t.Helper()
		expr, diags := hclsyntax.ParseExpression([]byte(src), "mock.tf", hcl.InitialPos)
		if diags.HasErrors() {
			t.Fatalf("invalid test expression %q: %s", src, diags.Error())
		}
		return expr
	}

	fromExpr := parseExpr(`test_instance.old[each.key]`)
	toExpr := parseExpr(`test_instance.new[each.key]`)
	forEachExpr := hcltest.MockExprTraversalSrc("local.moves")

	got, diags := decodeMovedBlock(&hcl.Block{
		Type: "moved",
		Body: hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"for_each": {Name: "for_each", Expr: forEachExpr},
				"from":     {Name: "from", Expr: fromExpr},
				"to":       {Name: "to", Expr: toExpr},
			},
		}),
		DefRange: blockRange,
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Error())
	}

	if got.From == nil || got.To == nil {
		t.Fatalf("expected parsed move endpoints")
	}
	if got.From.String() != "test_instance.old" {
		t.Fatalf("wrong shape from endpoint: %s", got.From)
	}
	if got.To.String() != "test_instance.new" {
		t.Fatalf("wrong shape to endpoint: %s", got.To)
	}
	if got.FromExpr == nil || got.ToExpr == nil || got.ForEach == nil {
		t.Fatal("expected raw expressions to be retained")
	}
}

func TestMovedBlock_inModule(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/valid-modules/moved-blocks")
	if diags.HasErrors() {
		t.Errorf("unexpected error: %s", diags.Error())
	}

	var gotPairs [][2]string
	for _, mc := range mod.Moved {
		gotPairs = append(gotPairs, [2]string{mc.From.String(), mc.To.String()})
	}
	wantPairs := [][2]string{
		{`test.foo`, `test.bar`},
		{`test.foo`, `test.bar["bloop"]`},
		{`module.a`, `module.b`},
		{`module.a`, `module.a["foo"]`},
		{`test.foo`, `module.a.test.foo`},
		{`data.test.foo`, `data.test.bar`},
	}
	if diff := cmp.Diff(wantPairs, gotPairs); diff != "" {
		t.Errorf("wrong addresses\n%s", diff)
	}
}

func mustMoveEndpointFromExpr(expr hcl.Expression) *addrs.MoveEndpoint {
	traversal, hcldiags := hcl.AbsTraversalForExpr(expr)
	if hcldiags.HasErrors() {
		panic(hcldiags.Errs())
	}

	ep, diags := addrs.ParseMoveEndpoint(traversal)
	if diags.HasErrors() {
		panic(diags.Err())
	}

	return ep
}

func exprTraversalEqual(a, b hcl.Expression) bool {
	switch {
	case a == nil || b == nil:
		return a == b
	}

	aTrav, aDiags := hcl.AbsTraversalForExpr(a)
	bTrav, bDiags := hcl.AbsTraversalForExpr(b)
	if aDiags.HasErrors() || bDiags.HasErrors() {
		return false
	}

	return reflect.DeepEqual(aTrav, bTrav)
}
