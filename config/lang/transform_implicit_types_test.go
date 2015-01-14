package lang

import (
	"testing"

	"github.com/hashicorp/terraform/config/lang/ast"
)

func TestLookupType(t *testing.T) {
	cases := []struct {
		Input  ast.Node
		Scope  *Scope
		Output ast.Type
		Error  bool
	}{
		{
			&customUntyped{},
			nil,
			ast.TypeInvalid,
			true,
		},

		{
			&customTyped{},
			nil,
			ast.TypeString,
			false,
		},

		{
			&ast.LiteralNode{
				Value: 42,
				Type:  ast.TypeInt,
			},
			nil,
			ast.TypeInt,
			false,
		},

		{
			&ast.VariableAccess{
				Name: "foo",
			},
			&Scope{
				VarMap: map[string]Variable{
					"foo": Variable{Type: ast.TypeInt},
				},
			},
			ast.TypeInt,
			false,
		},
	}

	for _, tc := range cases {
		actual, err := LookupType(tc.Input, tc.Scope)
		if (err != nil) != tc.Error {
			t.Fatalf("bad: %s\n\nInput: %#v", err, tc.Input)
		}
		if actual != tc.Output {
			t.Fatalf("bad: %s\n\nInput: %#v", actual, tc.Input)
		}
	}
}

type customUntyped struct{}

func (n customUntyped) Accept(ast.Visitor) {}
func (n customUntyped) Pos() (v ast.Pos)   { return }

type customTyped struct{}

func (n customTyped) Accept(ast.Visitor)            {}
func (n customTyped) Pos() (v ast.Pos)              { return }
func (n customTyped) Type(*Scope) (ast.Type, error) { return ast.TypeString, nil }
