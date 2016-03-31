package hilstructure

import (
	"reflect"
	"testing"

	"github.com/hashicorp/hil/ast"
)

func TestHILStringList_elements(t *testing.T) {
	expected := ast.Variable{
		Type: ast.TypeList,
		Value: []ast.Variable{
			ast.Variable{
				Type:  ast.TypeString,
				Value: "hello",
			},
			ast.Variable{
				Type:  ast.TypeString,
				Value: "world",
			},
		},
	}

	actual := MakeHILStringList([]string{"hello", "world"})

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected:\n\n%+v, Got:\n\n%+v\n", expected, actual)
	}
}

func TestHILStringList_empty(t *testing.T) {
	expected := ast.Variable{
		Type:  ast.TypeList,
		Value: []ast.Variable{},
	}

	actual := MakeHILStringList([]string{})

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected:\n\n%+v, Got:\n\n%+v\n", expected, actual)
	}
}
