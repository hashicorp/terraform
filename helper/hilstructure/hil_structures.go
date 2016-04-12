package hilstructure

import (
	"fmt"

	"github.com/hashicorp/hil/ast"
)

// MakeHILStringList returns an ast.Variable of type list, with
// a value equating to a list of ast.Variables each of type string,
// as is necessary for native list interpolation in HIL.
func MakeHILStringList(elements []string) ast.Variable {
	varElements := make([]ast.Variable, len(elements))
	for i, val := range elements {
		varElements[i] = ast.Variable{
			Type:  ast.TypeString,
			Value: val,
		}
	}

	return ast.Variable{
		Type:  ast.TypeList,
		Value: varElements,
	}
}

func HILStringListToSlice(input []ast.Variable) ([]string, error) {
	slice := make([]string, len(input))

	for i, variable := range input {
		if variable.Type != ast.TypeString {
			return nil, fmt.Errorf("Value at %d is not a string", i)
		}

		slice[i] = variable.Value.(string)
	}

	return slice, nil
}
