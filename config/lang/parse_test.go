package lang

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestParse(t *testing.T) {
	cases := []struct {
		Input  string
		Error  bool
		Result ast.Node
	}{
		{
			"foo",
			false,
			&ast.LiteralNode{
				Value: "foo",
				Type:  ast.TypeString,
			},
		},

		{
			"foo ${var.bar}",
			false,
			&ast.Concat{
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Type:  ast.TypeString,
					},
					&ast.VariableAccess{
						Name: "var.bar",
					},
				},
			},
		},

		{
			"foo ${\"bar\"}",
			false,
			&ast.Concat{
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Type:  ast.TypeString,
					},
					&ast.LiteralNode{
						Value: "bar",
						Type:  ast.TypeString,
					},
				},
			},
		},
	}

	for _, tc := range cases {
		actual, err := Parse(tc.Input)
		if (err != nil) != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("Bad: %#v\n\nInput: %s", actual, tc.Input)
		}
	}
}
