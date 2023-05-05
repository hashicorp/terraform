// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonfunction

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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
	ReturnType cty.Type `json:"return_type"`

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

func Marshal(f map[string]function.Function) ([]byte, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	signatures := newFunctions()

	for name, v := range f {
		if name == "can" {
			signatures.Signatures[name] = marshalCan(v)
		} else if name == "try" {
			signatures.Signatures[name] = marshalTry(v)
		} else {
			signature, err := marshalFunction(v)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					fmt.Sprintf("Failed to serialize function %q", name),
					err.Error(),
				))
			}
			signatures.Signatures[name] = signature
		}
	}

	if diags.HasErrors() {
		return nil, diags
	}

	ret, err := json.Marshal(signatures)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to serialize functions",
			err.Error(),
		))
		return nil, diags
	}
	return ret, nil
}

func marshalFunction(f function.Function) (*FunctionSignature, error) {
	var err error
	var vp *parameter
	if f.VarParam() != nil {
		vp = marshalParameter(f.VarParam())
	}

	var p []*parameter
	if len(f.Params()) > 0 {
		p = marshalParameters(f.Params())
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

// marshalTry returns a static function signature for the try function.
// We need this exception because the function implementation uses capsule
// types that we can't marshal.
func marshalTry(try function.Function) *FunctionSignature {
	return &FunctionSignature{
		Description: try.Description(),
		ReturnType:  cty.DynamicPseudoType,
		VariadicParameter: &parameter{
			Name:        try.VarParam().Name,
			Description: try.VarParam().Description,
			IsNullable:  try.VarParam().AllowNull,
			Type:        cty.DynamicPseudoType,
		},
	}
}

// marshalCan returns a static function signature for the can function.
// We need this exception because the function implementation uses capsule
// types that we can't marshal.
func marshalCan(can function.Function) *FunctionSignature {
	return &FunctionSignature{
		Description: can.Description(),
		ReturnType:  cty.Bool,
		Parameters: []*parameter{
			{
				Name:        can.Params()[0].Name,
				Description: can.Params()[0].Description,
				IsNullable:  can.Params()[0].AllowNull,
				Type:        cty.DynamicPseudoType,
			},
		},
	}
}
