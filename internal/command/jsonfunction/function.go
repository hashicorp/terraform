package jsonfunction

import (
	"encoding/json"

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
	ReturnType string `json:"return_type"`

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
		signatures.Signatures[k] = marshalFunction(v)
	}

	ret, err := json.Marshal(signatures)
	return ret, err
}

func marshalFunction(f function.Function) *FunctionSignature {
	var vp *parameter
	if f.VarParam() != nil {
		vp = marshalParameter(f.VarParam())
	}

	var p []*parameter
	if len(f.Params()) > 0 {
		p = marshalParameters(f.Params())
	}

	return &FunctionSignature{
		Description:       f.Description(),
		ReturnType:        getReturnType(f),
		Parameters:        p,
		VariadicParameter: vp,
	}
}
