package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcltest"
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestEvalValidateSelfRef(t *testing.T) {
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

			n := &EvalValidateSelfRef{
				Addr:   test.Addr,
				Config: body,
			}
			result, err := n.Eval(nil)
			if result != nil {
				t.Fatal("result should always be nil")
			}
			if (err != nil) != test.Err {
				t.Fatalf("err: %s", err)
			}
		})
	}
}
