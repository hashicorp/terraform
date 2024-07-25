// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestParseConfigResourceFromExpression(t *testing.T) {
	mustExpr := func(expr hcl.Expression, diags hcl.Diagnostics) hcl.Expression {
		if diags != nil {
			panic(diags.Error())
		}
		return expr
	}

	tests := []struct {
		expr   hcl.Expression
		expect addrs.ConfigResource
	}{
		{
			mustExpr(hclsyntax.ParseExpression([]byte("test_instance.bar"), "my_traversal", hcl.Pos{})),
			mustAbsResourceInstanceAddr("test_instance.bar").ConfigResource(),
		},

		// parsing should skip the each.key variable
		{
			mustExpr(hclsyntax.ParseExpression([]byte("test_instance.bar[each.key]"), "my_traversal", hcl.Pos{})),
			mustAbsResourceInstanceAddr("test_instance.bar").ConfigResource(),
		},

		// nested modules must work too
		{
			mustExpr(hclsyntax.ParseExpression([]byte("module.foo[each.key].test_instance.bar[each.key]"), "my_traversal", hcl.Pos{})),
			mustAbsResourceInstanceAddr("module.foo.test_instance.bar").ConfigResource(),
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, tc.expect), func(t *testing.T) {

			got, diags := parseConfigResourceFromExpression(tc.expr)
			if diags.HasErrors() {
				t.Fatal(diags.ErrWithWarnings())
			}
			if !got.Equal(tc.expect) {
				t.Fatalf("got %s, want %s", got, tc.expect)
			}
		})
	}
}

func TestImportBlock_decode(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
		End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
	}

	foo_str_expr := hcltest.MockExprLiteral(cty.StringVal("foo"))
	bar_expr := hcltest.MockExprTraversalSrc("test_instance.bar")

	bar_index_expr := hcltest.MockExprTraversalSrc("test_instance.bar[\"one\"]")

	mod_bar_expr := hcltest.MockExprTraversalSrc("module.bar.test_instance.bar")

	tests := map[string]struct {
		input *hcl.Block
		want  *Import
		err   string
	}{
		"success": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: foo_str_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ToResource: mustAbsResourceInstanceAddr("test_instance.bar").ConfigResource(),
				ID:         foo_str_expr,
				DeclRange:  blockRange,
			},
			``,
		},
		"indexed resources": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: foo_str_expr,
						},
						"to": {
							Name: "to",
							Expr: bar_index_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ToResource: mustAbsResourceInstanceAddr("test_instance.bar[\"one\"]").ConfigResource(),
				ID:         foo_str_expr,
				DeclRange:  blockRange,
			},
			``,
		},
		"resource inside module": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: foo_str_expr,
						},
						"to": {
							Name: "to",
							Expr: mod_bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ToResource: mustAbsResourceInstanceAddr("module.bar.test_instance.bar").ConfigResource(),
				ID:         foo_str_expr,
				DeclRange:  blockRange,
			},
			``,
		},
		"error: missing id argument": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ToResource: mustAbsResourceInstanceAddr("test_instance.bar").ConfigResource(),
				DeclRange:  blockRange,
			},
			"Missing required argument",
		},
		"error: missing to argument": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"to": {
							Name: "to",
							Expr: bar_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        foo_str_expr,
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: data source": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: foo_str_expr,
						},
						"to": {
							Name: "to",
							Expr: hcltest.MockExprTraversalSrc("data.test_instance.bar"),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        foo_str_expr,
				DeclRange: blockRange,
			},
			"Invalid import address",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeImportBlock(test.input)

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

			if diags.HasErrors() {
				return
			}

			if !got.ToResource.Equal(test.want.ToResource) {
				t.Errorf("expected resource %q got %q", test.want.ToResource, got.ToResource)
			}

			if !reflect.DeepEqual(got.ID, test.want.ID) {
				t.Errorf("expected ID %q got %q", test.want.ID, got.ID)
			}
		})
	}
}

func mustAbsResourceInstanceAddr(str string) addrs.AbsResourceInstance {
	addr, diags := addrs.ParseAbsResourceInstanceStr(str)
	if diags.HasErrors() {
		panic(fmt.Sprintf("invalid absolute resource instance address: %s", diags.Err()))
	}
	return addr
}
