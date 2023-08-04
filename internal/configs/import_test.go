// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

var (
	typeComparer  = cmp.Comparer(cty.Type.Equals)
	valueComparer = cmp.Comparer(cty.Value.RawEquals)
)

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
				To:        mustAbsResourceInstanceAddr("test_instance.bar"),
				ID:        foo_str_expr,
				DeclRange: blockRange,
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
				To:        mustAbsResourceInstanceAddr("test_instance.bar[\"one\"]"),
				ID:        foo_str_expr,
				DeclRange: blockRange,
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
				To:        mustAbsResourceInstanceAddr("module.bar.test_instance.bar"),
				ID:        foo_str_expr,
				DeclRange: blockRange,
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
				To:        mustAbsResourceInstanceAddr("test_instance.bar"),
				DeclRange: blockRange,
			},
			"Missing required argument",
		},
		"error: missing to argument": {
			&hcl.Block{
				Type: "import",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"id": {
							Name: "id",
							Expr: foo_str_expr,
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

			if !cmp.Equal(got, test.want, typeComparer, valueComparer) {
				t.Fatalf("wrong result: %s", cmp.Diff(got, test.want))
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
