package hil

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/mapstructure"
)

// UnknownValue is a sentinel value that can be used to denote
// that a value of a variable (or map element, list element, etc.)
// is unknown. This will always have the type ast.TypeUnknown.
const UnknownValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

var hilMapstructureDecodeHookSlice []interface{}
var hilMapstructureDecodeHookStringSlice []string
var hilMapstructureDecodeHookMap map[string]interface{}

// hilMapstructureWeakDecode behaves in the same way as mapstructure.WeakDecode
// but has a DecodeHook which defeats the backward compatibility mode of mapstructure
// which WeakDecodes []interface{}{} into an empty map[string]interface{}. This
// allows us to use WeakDecode (desirable), but not fail on empty lists.
func hilMapstructureWeakDecode(m interface{}, rawVal interface{}) error {
	config := &mapstructure.DecoderConfig{
		DecodeHook: func(source reflect.Type, target reflect.Type, val interface{}) (interface{}, error) {
			sliceType := reflect.TypeOf(hilMapstructureDecodeHookSlice)
			stringSliceType := reflect.TypeOf(hilMapstructureDecodeHookStringSlice)
			mapType := reflect.TypeOf(hilMapstructureDecodeHookMap)

			if (source == sliceType || source == stringSliceType) && target == mapType {
				return nil, fmt.Errorf("Cannot convert %s into a %s", source, target)
			}

			return val, nil
		},
		WeaklyTypedInput: true,
		Result:           rawVal,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(m)
}

// InterfaceToVariable takes a plain Go value and produces an ast.Variable representing
// the same value in HIL.
//
// It supports as input only strings, maps and lists, where maps and lists must themselves
// contain only strings, maps and lists. Maps and lists must also have consistent element
// types.
func InterfaceToVariable(input interface{}) (ast.Variable, error) {
	if inputVariable, ok := input.(ast.Variable); ok {
		return inputVariable, nil
	}

	var stringVal string
	if err := hilMapstructureWeakDecode(input, &stringVal); err == nil {
		// Special case the unknown value to turn into "unknown"
		if stringVal == UnknownValue {
			return ast.Variable{Value: UnknownValue, Type: ast.TypeUnknown}, nil
		}

		// Otherwise return the string value
		return ast.Variable{
			Type:  ast.TypeString,
			Value: stringVal,
		}, nil
	}

	var mapVal map[string]interface{}
	if err := hilMapstructureWeakDecode(input, &mapVal); err == nil {
		elements := make(map[string]ast.Variable)
		elemType := ast.TypeInvalid
		for i, element := range mapVal {
			varElement, err := InterfaceToVariable(element)
			if err != nil {
				return ast.Variable{}, err
			}
			elements[i] = varElement

			if varElement.Type == ast.TypeUnknown {
				// Ignore unknown values for the purposes of element type inference
				continue
			}

			if elemType == ast.TypeInvalid {
				elemType = varElement.Type
			} else if elemType != varElement.Type {
				return ast.Variable{}, fmt.Errorf(
					"inconsistent map element types; previously %s but found %s at key %s",
					elemType.Printable(), varElement.Type.Printable(), i,
				)
			}
		}

		// If the map is empty then we'll default to it being of type
		// string, just so we have some sort of element type.
		if elemType == ast.TypeInvalid {
			elemType = ast.TypeString
		}

		return ast.Variable{
			Type:  ast.TypeMap{elemType},
			Value: elements,
		}, nil
	}

	var sliceVal []interface{}
	if err := hilMapstructureWeakDecode(input, &sliceVal); err == nil {
		elements := make([]ast.Variable, len(sliceVal))
		elemType := ast.TypeInvalid
		for i, element := range sliceVal {
			varElement, err := InterfaceToVariable(element)
			if err != nil {
				return ast.Variable{}, err
			}
			elements[i] = varElement

			if varElement.Type == ast.TypeUnknown {
				// Ignore unknown values for the purposes of element type inference
				continue
			}

			if elemType == ast.TypeInvalid || elemType == ast.TypeUnknown {
				elemType = varElement.Type
			} else if elemType != varElement.Type {
				return ast.Variable{}, fmt.Errorf(
					"inconsistent list element types; previously %s but found %s at index %d",
					elemType.Printable(), varElement.Type.Printable(), i,
				)
			}
		}

		// If the map is empty then we'll default to it being of type
		// string, just so we have some sort of element type.
		if elemType == ast.TypeInvalid {
			elemType = ast.TypeString
		}

		return ast.Variable{
			Type:  ast.TypeList{elemType},
			Value: elements,
		}, nil
	}

	return ast.Variable{}, fmt.Errorf("value for conversion must be a string, []interface{} or map[string]interface: got %T", input)
}

// VariableToInterface takes an ast.Variable and produces a Go value that represents the same
// value within Go's type system.
//
// Only strings, maps and lists may be converted, and maps and lists must themselves have
// consistent elements types that are either strings, maps or lists. This matches the set
// of result types that can be produced by an ast.Output, so it should always be safe
// to pass an ast.Output result to this function.
func VariableToInterface(input ast.Variable) (interface{}, error) {
	if input.Type == ast.TypeString {
		if inputStr, ok := input.Value.(string); ok {
			return inputStr, nil
		} else {
			return nil, fmt.Errorf("ast.Variable with type string has value which is not a string")
		}
	}

	if ast.TypeIsList(input.Type) {
		inputList, ok := input.Value.([]ast.Variable)
		if !ok {
			return nil, fmt.Errorf("ast.Variable with type list has value which is not a []ast.Variable")
		}

		result := make([]interface{}, 0)
		if len(inputList) == 0 {
			return result, nil
		}

		for _, element := range inputList {
			if convertedElement, err := VariableToInterface(element); err == nil {
				result = append(result, convertedElement)
			} else {
				return nil, err
			}
		}

		return result, nil
	}

	if ast.TypeIsMap(input.Type) {
		inputMap, ok := input.Value.(map[string]ast.Variable)
		if !ok {
			return nil, fmt.Errorf("ast.Variable with type map has value which is not a map[string]ast.Variable")
		}

		result := make(map[string]interface{}, 0)
		if len(inputMap) == 0 {
			return result, nil
		}

		for key, value := range inputMap {
			if convertedValue, err := VariableToInterface(value); err == nil {
				result[key] = convertedValue
			} else {
				return nil, err
			}
		}

		return result, nil
	}

	return nil, fmt.Errorf("unknown input type: %s", input.Type)
}
