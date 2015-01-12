package lang

import (
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestTypeVisitor(t *testing.T) {
	cases := []struct {
		Input   string
		Visitor *TypeVisitor
		Error   bool
	}{
		{
			"foo",
			&TypeVisitor{},
			false,
		},

		{
			"foo ${bar}",
			&TypeVisitor{
				VarMap: map[string]Variable{
					"bar": Variable{
						Value: "baz",
						Type:  ast.TypeString,
					},
				},
			},
			false,
		},

		{
			"foo ${rand()}",
			&TypeVisitor{
				FuncMap: map[string]Function{
					"rand": Function{
						ReturnType: ast.TypeString,
						Callback: func([]interface{}) (interface{}, error) {
							return "42", nil
						},
					},
				},
			},
			false,
		},

		{
			"foo ${bar}",
			&TypeVisitor{
				VarMap: map[string]Variable{
					"bar": Variable{
						Value: 42,
						Type:  ast.TypeInt,
					},
				},
			},
			true,
		},
	}

	for _, tc := range cases {
		node, err := Parse(tc.Input)
		if err != nil {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}

		err = tc.Visitor.Visit(node)
		if (err != nil) != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
	}
}
