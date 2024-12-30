// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestValidateSelfRef(t *testing.T) {
	rAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "foo",
	}

	tests := []struct {
		Name string
		Addr addrs.Referenceable
		Expr hcl.Expression
		Err  bool
	}{
		{
			"no references at all",
			rAddr,
			hcltest.MockExprLiteral(cty.StringVal("bar")),
			false,
		},

		{
			"non self reference",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.bar.id"),
			false,
		},

		{
			"self reference",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.foo.id"),
			true,
		},

		{
			"self reference other index",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.foo[4].id"),
			false,
		},

		{
			"self reference same index",
			rAddr.Instance(addrs.IntKey(4)),
			hcltest.MockExprTraversalSrc("aws_instance.foo[4].id"),
			true,
		},

		{
			"self reference whole",
			rAddr.Instance(addrs.IntKey(4)),
			hcltest.MockExprTraversalSrc("aws_instance.foo"),
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			body := hcltest.MockBody(&hcl.BodyContent{
				Attributes: hcl.Attributes{
					"foo": {
						Name: "foo",
						Expr: test.Expr,
					},
				},
			})

			ps := providers.ProviderSchema{
				ResourceTypes: map[string]providers.Schema{
					"aws_instance": {
						Block: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"foo": {
									Type:     cty.String,
									Required: true,
								},
							},
						},
					},
				},
			}

			// First test the expression within the context of the configuration
			// body.
			diags := validateSelfRef(test.Addr, body, ps)
			if diags.HasErrors() != test.Err {
				if test.Err {
					t.Errorf("unexpected success; want error")
				} else {
					t.Errorf("unexpected error\n\n%s", diags.Err())
				}
			}
		})
	}
}

func TestValidateSelfInExpr(t *testing.T) {
	rAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "aws_instance",
		Name: "foo",
	}

	tests := []struct {
		Name string
		Addr addrs.Resource
		Expr hcl.Expression
		Err  bool
	}{
		{
			"no references at all",
			rAddr,
			hcltest.MockExprLiteral(cty.StringVal("bar")),
			false,
		},

		{
			"non self reference",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.bar.id"),
			false,
		},

		{
			"self reference",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.foo.id"),
			true,
		},

		{
			"self reference other index",
			rAddr,
			hcltest.MockExprTraversalSrc("aws_instance.foo[4].id"),
			true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.Name), func(t *testing.T) {
			diags := validateMetaSelfRef(test.Addr, test.Expr)
			if diags.HasErrors() != test.Err {
				if test.Err {
					t.Errorf("unexpected success; want error")
				} else {
					t.Errorf("unexpected error\n\n%s", diags.Err())
				}
			}
		})
	}
}
