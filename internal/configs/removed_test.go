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
				From:      mustRemoveEndpointFromExpr(foo_expr),
				Destroy:   true,
				Managed:   &ManagedResource{},
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
				From:      mustRemoveEndpointFromExpr(foo_expr),
				Destroy:   false,
				Managed:   &ManagedResource{},
				DeclRange: blockRange,
			},
			``,
		},
		"provisioner when = destroy": {
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
							Type:        "provisioner",
							Labels:      []string{"remote-exec"},
							LabelRanges: []hcl.Range{{}},
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"when": {
										Name: "when",
										Expr: hcltest.MockExprTraversalSrc("destroy"),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:    mustRemoveEndpointFromExpr(foo_expr),
				Destroy: true,
				Managed: &ManagedResource{
					Provisioners: []*Provisioner{
						{
							Type: "remote-exec",
							Config: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{},
								Blocks:     hcl.Blocks{},
							}),
							When:      ProvisionerWhenDestroy,
							OnFailure: ProvisionerOnFailureFail,
						},
					},
				},
				DeclRange: blockRange,
			},
			``,
		},
		"provisioner when = create": {
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
							Type:        "provisioner",
							Labels:      []string{"local-exec"},
							LabelRanges: []hcl.Range{{}},
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"when": {
										Name: "when",
										Expr: hcltest.MockExprTraversalSrc("create"),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:    mustRemoveEndpointFromExpr(foo_expr),
				Destroy: true,
				Managed: &ManagedResource{
					Provisioners: []*Provisioner{
						{
							Type: "local-exec",
							Config: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{},
								Blocks:     hcl.Blocks{},
							}),
							When:      ProvisionerWhenCreate,
							OnFailure: ProvisionerOnFailureFail,
						},
					},
				},
				DeclRange: blockRange,
			},
			`Invalid provisioner block`,
		},
		"provisioner no when": {
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
							Type: "connection",
							Body: hcltest.MockBody(&hcl.BodyContent{}),
						},
						&hcl.Block{
							Type:        "provisioner",
							Labels:      []string{"local-exec"},
							LabelRanges: []hcl.Range{{}},
							Body:        hcltest.MockBody(&hcl.BodyContent{}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:    mustRemoveEndpointFromExpr(foo_expr),
				Destroy: true,
				Managed: &ManagedResource{
					Connection: &Connection{
						Config: hcltest.MockBody(&hcl.BodyContent{}),
					},
					Provisioners: []*Provisioner{
						{
							Type: "local-exec",
							Config: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{},
								Blocks:     hcl.Blocks{},
							}),
							When:      ProvisionerWhenCreate,
							OnFailure: ProvisionerOnFailureFail,
						},
					},
				},
				DeclRange: blockRange,
			},
			`Invalid provisioner block`,
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
				From:      mustRemoveEndpointFromExpr(mod_foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			``,
		},
		"provisioner for module": {
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
							Type:        "provisioner",
							Labels:      []string{"local-exec"},
							LabelRanges: []hcl.Range{{}},
							Body: hcltest.MockBody(&hcl.BodyContent{
								Attributes: hcl.Attributes{
									"when": {
										Name: "when",
										Expr: hcltest.MockExprTraversalSrc("destroy"),
									},
								},
							}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustRemoveEndpointFromExpr(mod_foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			`Invalid provisioner block`,
		},
		"connection for module": {
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
							Type: "connection",
							Body: hcltest.MockBody(&hcl.BodyContent{}),
						},
					},
				}),
				DefRange: blockRange,
			},
			&Removed{
				From:      mustRemoveEndpointFromExpr(mod_foo_expr),
				Destroy:   true,
				DeclRange: blockRange,
			},
			`Invalid connection block`,
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
				From:      mustRemoveEndpointFromExpr(foo_expr),
				Destroy:   true,
				Managed:   &ManagedResource{},
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
				From:      nil,
				Destroy:   true,
				DeclRange: blockRange,
			},
			`Resource instance keys not allowed`,
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
				From:      nil,
				Destroy:   true,
				DeclRange: blockRange,
			},
			`Module instance keys not allowed`,
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

func mustRemoveEndpointFromExpr(expr hcl.Expression) *addrs.RemoveTarget {
	traversal, hcldiags := hcl.AbsTraversalForExpr(expr)
	if hcldiags.HasErrors() {
		panic(hcldiags.Errs())
	}

	ep, diags := addrs.ParseRemoveTarget(traversal)
	if diags.HasErrors() {
		panic(diags.Err())
	}

	return ep
}
