package jsonfunction

import (
	"encoding/json"
	"fmt"

	"github.com/zclconf/go-cty/cty/function"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "1.0"

// functions is the top-level object returned when exporting function signatures
type functions struct {
	FormatVersion string                        `json:"format_version"`
	Signatures    map[string]*FunctionSignature `json:"function_signatures,omitempty"`
}

// FunctionSignature represents a function signature.
type FunctionSignature struct {
	// Description is an optional human-readable description
	// of the function
	Description string `json:"description,omitempty"`

	// ReturnTypes is the ctyjson representation of the function's
	// return types based on supplying all parameters using
	// dynamic types. Functions can have dynamic return types.
	// TODO? could we use cty.Type here instead of calling ctyjson.MarshalType manually?
	// TODO? see: https://github.com/zclconf/go-cty/blob/main/cty/json/type.go
	ReturnType json.RawMessage `json:"return_type"`

	// Parameters describes the function's fixed positional parameters.
	Parameters []*parameter `json:"parameters,omitempty"`

	// VariadicParameter describes the function's variadic
	// parameters, if any are supported.
	VariadicParameter *parameter `json:"variadic_parameter,omitempty"`
}

func newFunctions() *functions {
	signatures := make(map[string]*FunctionSignature)
	return &functions{
		FormatVersion: FormatVersion,
		Signatures:    signatures,
	}
}

func Marshal(f map[string]function.Function) ([]byte, error) {
	signatures := newFunctions()

	for k, v := range f {
		signature, err := marshalFunction(v)
		if err != nil {
			fmt.Printf("Failed to serialize function %q: %s\n", k, err) // TODO! handle error
			continue
		}
		signatures.Signatures[k] = signature
	}

	ret, err := json.Marshal(signatures)
	return ret, err
}

func marshalFunction(f function.Function) (*FunctionSignature, error) {
	var err error
	var vp *parameter
	if f.VarParam() != nil {
		vp, err = marshalParameter(f.VarParam())
		if err != nil {
			return nil, err
		}
	}

	var p []*parameter
	if len(f.Params()) > 0 {
		p, err = marshalParameters(f.Params())
		if err != nil {
			return nil, err
		}
	}

	r, err := getReturnType(f)
	if err != nil {
		return nil, err
	}

	return &FunctionSignature{
		Description:       f.Description(),
		ReturnType:        r,
		Parameters:        p,
		VariadicParameter: vp,
	}, nil
}
