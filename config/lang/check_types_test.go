package lang

import (
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestTypeCheck(t *testing.T) {
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
			"foo ${bar}",
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
			"foo ${rand()}",
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
			`foo ${rand("42")}`,
			&Scope{
				FuncMap: map[string]Function{
					"rand": Function{
						ArgTypes:   []ast.Type{ast.TypeString},
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
			`foo ${rand(42)}`,
			&Scope{
				FuncMap: map[string]Function{
					"rand": Function{
						ArgTypes:   []ast.Type{ast.TypeString},
						ReturnType: ast.TypeString,
						Callback: func([]interface{}) (interface{}, error) {
							return "42", nil
						},
					},
				},
			},
			true,
		},

		{
			"foo ${bar}",
			&Scope{
				VarMap: map[string]Variable{
					"bar": Variable{
						Value: 42,
						Type:  ast.TypeInt,
					},
				},
			},
			true,
		},

		{
			"foo ${rand()}",
			&Scope{
				FuncMap: map[string]Function{
					"rand": Function{
						ReturnType: ast.TypeInt,
						Callback: func([]interface{}) (interface{}, error) {
							return 42, nil
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

		visitor := &TypeCheck{Scope: tc.Scope}
		err = visitor.Visit(node)
		if (err != nil) != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
	}
}
