// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonfunction

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
//
// Any changes to this version should also consider compatibility in the
// jsonprovider package versioning as well, as that functionality is also
// reliant on this package.
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

	// Summary is the optional shortened description of the function
	Summary string `json:"summary,omitempty"`

	// DeprecationMessage is an optional message that indicates that the
	// function should be considered deprecated and what actions should be
	// performed by the practitioner to handle the deprecation.
	DeprecationMessage string `json:"deprecation_message,omitempty"`

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

func MarshalProviderFunctions(f map[string]providers.FunctionDecl) map[string]*FunctionSignature {
	if f == nil {
		return nil
	}

	result := make(map[string]*FunctionSignature, len(f))

	for name, v := range f {
		result[name] = marshalProviderFunction(v)
	}

	return result
}

func Marshal(f map[string]function.Function) ([]byte, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	signatures := newFunctions()

	for name, v := range f {
		if name == "can" || name == "core::can" {
			signatures.Signatures[name] = marshalCan(v)
		} else if name == "try" || name == "core::try" {
			signatures.Signatures[name] = marshalTry(v)
		} else if name == "templatestring" || name == "core::templatestring" {
			signatures.Signatures[name] = marshalTemplatestring(v)
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

func marshalProviderFunction(f providers.FunctionDecl) *FunctionSignature {
	var vp *parameter
	if f.VariadicParameter != nil {
		vp = marshalProviderParameter(*f.VariadicParameter)
	}

	var p []*parameter
	if len(f.Parameters) > 0 {
		p = marshalProviderParameters(f.Parameters)
	}

	return &FunctionSignature{
		Description:        f.Description,
		Summary:            f.Summary,
		DeprecationMessage: f.DeprecationMessage,
		ReturnType:         f.ReturnType,
		Parameters:         p,
		VariadicParameter:  vp,
	}
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

// marshalTemplatestring returns a static function signature for the
// templatestring function.
// We need this exception because the function implementation uses capsule
// types that we can't marshal.
func marshalTemplatestring(templatestring function.Function) *FunctionSignature {
	return &FunctionSignature{
		Description: templatestring.Description(),
		ReturnType:  cty.String,
		Parameters: []*parameter{
			{
				Name:        templatestring.Params()[0].Name,
				Description: templatestring.Params()[0].Description,
				IsNullable:  templatestring.Params()[0].AllowNull,
				Type:        cty.String,
			},
			{
				Name:        templatestring.Params()[1].Name,
				Description: templatestring.Params()[1].Description,
				IsNullable:  templatestring.Params()[1].AllowNull,
				Type:        cty.DynamicPseudoType,
			},
		},
	}
}
