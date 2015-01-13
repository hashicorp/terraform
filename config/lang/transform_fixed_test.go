package lang

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestFixedValueTransform(t *testing.T) {
	cases := []struct {
		Input  ast.Node
		Output ast.Node
	}{
		{
			&ast.LiteralNode{Value: 42},
			&ast.LiteralNode{Value: 42},
		},

		{
			&ast.VariableAccess{Name: "bar"},
			&ast.LiteralNode{Value: "foo"},
		},

		{
			&ast.Concat{
				Exprs: []ast.Node{
					&ast.VariableAccess{Name: "bar"},
					&ast.LiteralNode{Value: 42},
				},
			},
			&ast.Concat{
				Exprs: []ast.Node{
					&ast.LiteralNode{Value: "foo"},
					&ast.LiteralNode{Value: 42},
				},
			},
		},
	}

	value := &ast.LiteralNode{Value: "foo"}
	for _, tc := range cases {
		actual := FixedValueTransform(tc.Input, value)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("bad: %#v\n\nInput: %#v", actual, tc.Input)
		}
	}
}
