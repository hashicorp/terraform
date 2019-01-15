package jsonconfig

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/hashicorp/terraform/configs/configschema"
)

func TestMarshalExpressions(t *testing.T) {
	tests := []struct {
		Input  hcl.Body
		Schema *configschema.Block
		Want   expressions
	}{
		{
			&hclsyntax.Body{
				Attributes: hclsyntax.Attributes{
					"foo": &hclsyntax.Attribute{
						Expr: &hclsyntax.LiteralValueExpr{
							Val: cty.StringVal("bar"),
						},
					},
				},
			},
			&configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
			expressions{
				"foo": expression{
					ConstantValue: json.RawMessage([]byte(`"bar"`)),
					References:    []string(nil),
				},
			},
		},
	}

	for _, test := range tests {
		got := marshalExpressions(test.Input, test.Schema)
		if !reflect.DeepEqual(got, test.Want) {
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

func TestMarshalExpression(t *testing.T) {
	tests := []struct {
		Input hcl.Expression
		Want  expression
	}{
		{
			nil,
			expression{},
		},
	}

	for _, test := range tests {
		got := marshalExpression(test.Input)
		if !reflect.DeepEqual(got, test.Want) {
			t.Fatalf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}
