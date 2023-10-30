// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
)

func TestRemovedBlock_decode(t *testing.T) {
	blockRange := hcl.Range{
		Filename: "mock.tf",
		Start:    hcl.Pos{Line: 3, Column: 12, Byte: 27},
		End:      hcl.Pos{Line: 3, Column: 19, Byte: 34},
	}

	foo_expr := hcltest.MockExprTraversalSrc("test_instance.foo")
	foo_index_expr := hcltest.MockExprTraversalSrc("test_instance.foo[1]")
	mod_foo_expr := hcltest.MockExprTraversalSrc("module.foo")
	mod_foo_index_expr := hcltest.MockExprTraversalSrc("module.foo[1]")

	tests := map[string]struct {
		input *hcl.Block
		want  *Removed
		err   string
	}{
		"destroy true": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
					},
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(true)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustMoveEndpointFromExpr(foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			``,
		},
		"destroy false": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_expr,
						},
					},
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(false)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustMoveEndpointFromExpr(foo_expr),
				Destroy:   false,
				DeclRange: blockRange,
			},
			``,
		},
		"modules": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: mod_foo_expr,
						},
					},
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(true)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustMoveEndpointFromExpr(mod_foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			``,
		},
		// KEM Unspecified behaviour
		"no lifecycle block": {
			&hcl.Block{
				Type: "removed",
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
			&Removed{
				From:      mustMoveEndpointFromExpr(foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			``,
		},
		"error: missing argument": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(true)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				Destroy:   true,
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: indexed resource instance": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: foo_index_expr,
						},
					},
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(true)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustMoveEndpointFromExpr(foo_index_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			`No`,
		},
		"error: indexed module instance": {
			&hcl.Block{
				Type: "removed",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"from": {
							Name: "from",
							Expr: mod_foo_index_expr,
						},
					},
					Blocks: hcl.Blocks{
						&hcl.Block{
							Type: "lifecycle",
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"destroy": {
										Name: "destroy",
										Expr: hcltest.MockExprLiteral(cty.BoolVal(true)),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustMoveEndpointFromExpr(mod_foo_index_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			`No`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, diags := decodeRemovedBlock(test.input)

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

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(addrs.MoveEndpoint{})) {
				t.Fatalf("wrong result: %s", cmp.Diff(got, test.want))
			}
		})
	}
}
