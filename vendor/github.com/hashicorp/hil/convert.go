package hil

import (
	"fmt"

	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/mapstructure"
)

func InterfaceToVariable(input interface{}) (ast.Variable, error) {
	var stringVal string
	if err := mapstructure.WeakDecode(input, &stringVal); err == nil {
		return ast.Variable{
			Type:  ast.TypeString,
			Value: stringVal,
		}, nil
	}

	var sliceVal []interface{}
	if err := mapstructure.WeakDecode(input, &sliceVal); err == nil {
		elements := make([]ast.Variable, len(sliceVal))
		for i, element := range sliceVal {
			varElement, err := InterfaceToVariable(element)
			if err != nil {
				return ast.Variable{}, err
			}
			elements[i] = varElement
		}

		return ast.Variable{
			Type:  ast.TypeList,
			Value: elements,
		}, nil
	}

	var mapVal map[string]interface{}
	if err := mapstructure.WeakDecode(input, &mapVal); err == nil {
		elements := make(map[string]ast.Variable)
		for i, element := range mapVal {
			varElement, err := InterfaceToVariable(element)
			if err != nil {
				return ast.Variable{}, err
			}
			elements[i] = varElement
		}

		return ast.Variable{
			Type:  ast.TypeMap,
			Value: elements,
		}, nil
	}

	return ast.Variable{}, fmt.Errorf("value for conversion must be a string, interface{} or map[string]interface: got %T", input)
}
