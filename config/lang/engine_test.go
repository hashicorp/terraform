package lang

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestEngineExecute(t *testing.T) {
	cases := []struct {
		Input      string
		Engine     *Engine
		Error      bool
		Result     interface{}
		ResultType ast.Type
	}{
		{
			"foo",
			&Engine{},
			false,
			"foo",
			ast.TypeString,
		},

		{
			"foo ${bar}",
			&Engine{
				VarMap: map[string]Variable{
					"bar": Variable{
						Value: "baz",
						Type:  ast.TypeString,
					},
				},
			},
			false,
			"foo baz",
			ast.TypeString,
		},

		{
			"foo ${rand()}",
			&Engine{
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
			"foo 42",
			ast.TypeString,
		},
	}

	for _, tc := range cases {
		node, err := Parse(tc.Input)
		if err != nil {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}

		out, outType, err := tc.Engine.Execute(node)
		if (err != nil) != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
		if outType != tc.ResultType {
			t.Fatalf("Bad: %s\n\nInput: %s", outType, tc.Input)
		}
		if !reflect.DeepEqual(out, tc.Result) {
			t.Fatalf("Bad: %#v\n\nInput: %s", out, tc.Input)
		}
	}
}
