package config

import (
	"fmt"

	"github.com/hashicorp/hil/ast"
)

// stringSliceToVariableValue converts a string slice into the value
// required to be returned from interpolation functions which return
// TypeList.
func stringSliceToVariableValue(values []string) []ast.Variable {
	output := make([]ast.Variable, len(values))
	for index, value := range values {
		output[index] = ast.Variable{
			Type:  ast.TypeString,
			Value: value,
		}
	}
	return output
}

// listVariableSliceToVariableValue converts a list of lists into the value
// required to be returned from interpolation functions which return TypeList.
func listVariableSliceToVariableValue(values [][]ast.Variable) []ast.Variable {
	output := make([]ast.Variable, len(values))

	for index, value := range values {
		output[index] = ast.Variable{
			Type:  ast.TypeList,
			Value: value,
		}
	}
	return output
}

func listVariableValueToStringSlice(values []ast.Variable) ([]string, error) {
	output := make([]string, len(values))
	for index, value := range values {
		if value.Type != ast.TypeString {
			return []string{}, fmt.Errorf("list has non-string element (%T)", value.Type.String())
		}
		output[index] = value.Value.(string)
	}
	return output, nil
}

// Funcs used to return a mapping of built-in functions for configuration.
//
// However, these function implementations are no longer used. To find the
// current function implementations, refer to ../lang/functions.go  instead.
func Funcs() map[string]ast.Function {
	return nil
}
