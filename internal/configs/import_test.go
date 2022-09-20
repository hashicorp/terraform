package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestImportBlock_decode(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
		End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
	}

	id := "test1"
	id_expr := hcltest.MockExprLiteral(cty.StringVal(id))

	res_expr := hcltest.MockExprTraversalSrc("test_instance.foo")

	index_expr := hcltest.MockExprTraversalSrc("test_instance.foo[1]")

	mod_expr := hcltest.MockExprTraversalSrc("module.foo.test_instance.this")

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
							Expr: id_expr,
						},
						"to": {
							Name: "to",
							Expr: res_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        id,
				To:        mustImportEndpointFromExpr(res_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"indexed_resource": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: id_expr,
						},
						"to": {
							Name: "to",
							Expr: index_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        id,
				To:        mustImportEndpointFromExpr(index_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"module": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: id_expr,
						},
						"to": {
							Name: "to",
							Expr: mod_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        id,
				To:        mustImportEndpointFromExpr(mod_expr),
				DeclRange: blockRange,
			},
			``,
		},
		"error: missing argument": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: id_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				ID:        id,
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: type mismatch": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
						"to": {
							Name: "to",
							Expr: res_expr,
						},
					},
				}),
				DefRange: blockRange,
			},
			&Import{
				To:        mustImportEndpointFromExpr(res_expr),
				DeclRange: blockRange,
			},
			"Invalid Attribute",
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
				t.Fatalf("expected error")
			}
			if !cmp.Equal(got, test.want, cmp.AllowUnexported(addrs.AbsResourceInstance{})) {
				t.Fatalf("wrong result: %s", cmp.Diff(got, test.want))
			}
		})
	}
}

func mustImportEndpointFromExpr(expr hcl.Expression) addrs.AbsResourceInstance {
	traversal, hcldiags := hcl.AbsTraversalForExpr(expr)
	if hcldiags.HasErrors() {
		panic(hcldiags.Errs())
	}
	ep, diags := addrs.ParseAbsResourceInstance(traversal)
	if diags.HasErrors() {
		panic(diags.Err())
	}
	return ep
}
