package lang

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestEval(t *testing.T) {
	cases := []struct {
		Input      string
		Scope      *ast.BasicScope
		Error      bool
		Result     interface{}
		ResultType ast.Type
	}{
		{
			"foo",
			nil,
			false,
			"foo",
			ast.TypeString,
		},

		{
			"foo ${bar}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
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
			"foo ${42+1}",
			nil,
			false,
			"foo 43",
			ast.TypeString,
		},

		{
			"foo ${42-1}",
			nil,
			false,
			"foo 41",
			ast.TypeString,
		},

		{
			"foo ${42*2}",
			nil,
			false,
			"foo 84",
			ast.TypeString,
		},

		{
			"foo ${42/2}",
			nil,
			false,
			"foo 21",
			ast.TypeString,
		},

		{
			"foo ${42%4}",
			nil,
			false,
			"foo 2",
			ast.TypeString,
		},

		{
			"foo ${42.0+1.0}",
			nil,
			false,
			"foo 43",
			ast.TypeString,
		},

		{
			"foo ${42.0+1}",
			nil,
			false,
			"foo 43",
			ast.TypeString,
		},

		{
			"foo ${42+1.0}",
			nil,
			false,
			"foo 43",
			ast.TypeString,
		},

		{
			"foo ${42+2*2}",
			nil,
			false,
			"foo 88",
			ast.TypeString,
		},

		{
			"foo ${42+(2*2)}",
			nil,
			false,
			"foo 46",
			ast.TypeString,
		},

		{
			"foo ${bar+1}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: 41,
						Type:  ast.TypeInt,
					},
				},
			},
			false,
			"foo 42",
			ast.TypeString,
		},

		{
			"foo ${bar+1}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: "41",
						Type:  ast.TypeString,
					},
				},
			},
			false,
			"foo 42",
			ast.TypeString,
		},

		{
			"foo ${bar+baz}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: "41",
						Type:  ast.TypeString,
					},
					"baz": ast.Variable{
						Value: "1",
						Type:  ast.TypeString,
					},
				},
			},
			false,
			"foo 42",
			ast.TypeString,
		},

		{
			"foo ${rand()}",
			&ast.BasicScope{
				FuncMap: map[string]ast.Function{
					"rand": ast.Function{
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

		{
			`foo ${rand("foo", "bar")}`,
			&ast.BasicScope{
				FuncMap: map[string]ast.Function{
					"rand": ast.Function{
						ReturnType:   ast.TypeString,
						Variadic:     true,
						VariadicType: ast.TypeString,
						Callback: func(args []interface{}) (interface{}, error) {
							var result string
							for _, a := range args {
								result += a.(string)
							}
							return result, nil
						},
					},
				},
			},
			false,
			"foo foobar",
			ast.TypeString,
		},

		// Testing implicit type conversions

		{
			"foo ${bar}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: 42,
						Type:  ast.TypeInt,
					},
				},
			},
			false,
			"foo 42",
			ast.TypeString,
		},

		{
			`foo ${foo("42")}`,
			&ast.BasicScope{
				FuncMap: map[string]ast.Function{
					"foo": ast.Function{
						ArgTypes:   []ast.Type{ast.TypeInt},
						ReturnType: ast.TypeString,
						Callback: func(args []interface{}) (interface{}, error) {
							return strconv.FormatInt(int64(args[0].(int)), 10), nil
						},
					},
				},
			},
			false,
			"foo 42",
			ast.TypeString,
		},

		// Multiline
		{
			"foo ${42+\n1.0}",
			nil,
			false,
			"foo 43",
			ast.TypeString,
		},

		{
			"foo ${-46}",
			nil,
			false,
			"foo -46",
			ast.TypeString,
		},

		{
			"foo ${-46 + 5}",
			nil,
			false,
			"foo -41",
			ast.TypeString,
		},

		{
			"foo ${46 + -5}",
			nil,
			false,
			"foo 41",
			ast.TypeString,
		},

		{
			"foo ${-bar}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: 41,
						Type:  ast.TypeInt,
					},
				},
			},
			false,
			"foo -41",
			ast.TypeString,
		},

		{
			"foo ${5 + -bar}",
			&ast.BasicScope{
				VarMap: map[string]ast.Variable{
					"bar": ast.Variable{
						Value: 41,
						Type:  ast.TypeInt,
					},
				},
			},
			false,
			"foo -36",
			ast.TypeString,
		},
	}

	for _, tc := range cases {
		node, err := Parse(tc.Input)
		if err != nil {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}

		out, outType, err := Eval(node, &EvalConfig{GlobalScope: tc.Scope})
		if err != nil != tc.Error {
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
