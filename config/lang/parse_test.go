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
			"",
			false,
			&ast.LiteralNode{
				Value: "",
				Typex: ast.TypeString,
				Posx:  ast.Pos{Column: 1, Line: 1},
			},
		},

		{
			"foo",
			false,
			&ast.LiteralNode{
				Value: "foo",
				Typex: ast.TypeString,
				Posx:  ast.Pos{Column: 1, Line: 1},
			},
		},

		{
			"$${var.foo}",
			false,
			&ast.LiteralNode{
				Value: "${var.foo}",
				Typex: ast.TypeString,
				Posx:  ast.Pos{Column: 1, Line: 1},
			},
		},

		{
			"foo ${var.bar}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.VariableAccess{
						Name: "var.bar",
						Posx: ast.Pos{Column: 7, Line: 1},
					},
				},
			},
		},

		{
			"foo ${var.bar} baz",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.VariableAccess{
						Name: "var.bar",
						Posx: ast.Pos{Column: 7, Line: 1},
					},
					&ast.LiteralNode{
						Value: " baz",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 15, Line: 1},
					},
				},
			},
		},

		{
			"foo ${\"bar\"}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.LiteralNode{
						Value: "bar",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 7, Line: 1},
					},
				},
			},
		},

		{
			`foo ${func('baz')}`,
			true,
			nil,
		},

		{
			"foo ${42}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.LiteralNode{
						Value: 42,
						Typex: ast.TypeInt,
						Posx:  ast.Pos{Column: 7, Line: 1},
					},
				},
			},
		},

		{
			"foo ${3.14159}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.LiteralNode{
						Value: 3.14159,
						Typex: ast.TypeFloat,
						Posx:  ast.Pos{Column: 7, Line: 1},
					},
				},
			},
		},

		{
			"foo ${42+1}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.Arithmetic{
						Op: ast.ArithmeticOpAdd,
						Exprs: []ast.Node{
							&ast.LiteralNode{
								Value: 42,
								Typex: ast.TypeInt,
								Posx:  ast.Pos{Column: 7, Line: 1},
							},
							&ast.LiteralNode{
								Value: 1,
								Typex: ast.TypeInt,
								Posx:  ast.Pos{Column: 10, Line: 1},
							},
						},
						Posx: ast.Pos{Column: 7, Line: 1},
					},
				},
			},
		},

		{
			"foo ${var.bar*1} baz",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.Arithmetic{
						Op: ast.ArithmeticOpMul,
						Exprs: []ast.Node{
							&ast.VariableAccess{
								Name: "var.bar",
								Posx: ast.Pos{Column: 7, Line: 1},
							},
							&ast.LiteralNode{
								Value: 1,
								Typex: ast.TypeInt,
								Posx:  ast.Pos{Column: 15, Line: 1},
							},
						},
						Posx: ast.Pos{Column: 7, Line: 1},
					},
					&ast.LiteralNode{
						Value: " baz",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 17, Line: 1},
					},
				},
			},
		},

		{
			"${foo()}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 3, Line: 1},
				Exprs: []ast.Node{
					&ast.Call{
						Func: "foo",
						Args: nil,
						Posx: ast.Pos{Column: 3, Line: 1},
					},
				},
			},
		},

		{
			"${foo(bar)}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 3, Line: 1},
				Exprs: []ast.Node{
					&ast.Call{
						Func: "foo",
						Posx: ast.Pos{Column: 3, Line: 1},
						Args: []ast.Node{
							&ast.VariableAccess{
								Name: "bar",
								Posx: ast.Pos{Column: 7, Line: 1},
							},
						},
					},
				},
			},
		},

		{
			"${foo(bar, baz)}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 3, Line: 1},
				Exprs: []ast.Node{
					&ast.Call{
						Func: "foo",
						Posx: ast.Pos{Column: 3, Line: 1},
						Args: []ast.Node{
							&ast.VariableAccess{
								Name: "bar",
								Posx: ast.Pos{Column: 7, Line: 1},
							},
							&ast.VariableAccess{
								Name: "baz",
								Posx: ast.Pos{Column: 11, Line: 1},
							},
						},
					},
				},
			},
		},

		{
			"${foo(bar(baz))}",
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 3, Line: 1},
				Exprs: []ast.Node{
					&ast.Call{
						Func: "foo",
						Posx: ast.Pos{Column: 3, Line: 1},
						Args: []ast.Node{
							&ast.Call{
								Func: "bar",
								Posx: ast.Pos{Column: 7, Line: 1},
								Args: []ast.Node{
									&ast.VariableAccess{
										Name: "baz",
										Posx: ast.Pos{Column: 11, Line: 1},
									},
								},
							},
						},
					},
				},
			},
		},

		{
			`foo ${"bar ${baz}"}`,
			false,
			&ast.Concat{
				Posx: ast.Pos{Column: 1, Line: 1},
				Exprs: []ast.Node{
					&ast.LiteralNode{
						Value: "foo ",
						Typex: ast.TypeString,
						Posx:  ast.Pos{Column: 1, Line: 1},
					},
					&ast.Concat{
						Posx: ast.Pos{Column: 7, Line: 1},
						Exprs: []ast.Node{
							&ast.LiteralNode{
								Value: "bar ",
								Typex: ast.TypeString,
								Posx:  ast.Pos{Column: 7, Line: 1},
							},
							&ast.VariableAccess{
								Name: "baz",
								Posx: ast.Pos{Column: 14, Line: 1},
							},
						},
					},
				},
			},
		},

		{
			`foo ${bar ${baz}}`,
			true,
			nil,
		},

		{
			`foo ${${baz}}`,
			true,
			nil,
		},

		{
			"${var",
			true,
			nil,
		},
	}

	for _, tc := range cases {
		actual, err := Parse(tc.Input)
		if err != nil != tc.Error {
			t.Fatalf("Error: %s\n\nInput: %s", err, tc.Input)
		}
		if !reflect.DeepEqual(actual, tc.Result) {
			t.Fatalf("Bad: %#v\n\nInput: %s", actual, tc.Input)
		}
	}
}
