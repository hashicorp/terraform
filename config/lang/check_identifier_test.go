package lang

import (
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestIdentifierCheck(t *testing.T) {
	cases := []struct {
		Input string
		Scope *Scope
		Error bool
	}{
		{
			"foo",
			&Scope{},
			false,
		},

		{
			"foo ${bar} success",
			&Scope{
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
			"foo ${bar}",
			&Scope{},
			true,
		},

		{
			"foo ${rand()} success",
			&Scope{
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
			"foo ${rand()}",
			&Scope{},
			true,
		},

		{
			"foo ${rand(42)} ",
			&Scope{
				FuncMap: map[string]Function{
					"rand": Function{
						ReturnType: ast.TypeString,
						Callback: func([]interface{}) (interface{}, error) {
							return "42", nil
						},
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

		visitor := &IdentifierCheck{Scope: tc.Scope}
		err = visitor.Visit(node)
		if (err != nil) != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
	}
}
