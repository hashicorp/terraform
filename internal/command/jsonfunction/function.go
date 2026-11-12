// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package jsonfunction

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// FunctionSignature represents a function signature.
//
// Any changes to this struct will impact versioning in the jsonprovider package's versioning,
// in addition to versioning for the `metadata functions` command's JSON's output
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

func MarshalToFunctionSignatures(f map[string]function.Function) (map[string]*FunctionSignature, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	signatures := make(map[string]*FunctionSignature, len(f))

	for name, v := range f {
		switch name {
		case "can", "core::can":
			signatures[name] = marshalCan(v)
		case "try", "core::try":
			signatures[name] = marshalTry(v)
		case "templatestring", "core::templatestring":
			signatures[name] = marshalTemplatestring(v)
		case "convert", "core::convert":
			signatures[name] = marshalConvert(v)
		default:
			signature, err := marshalFunction(v)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					fmt.Sprintf("Failed to serialize function %q", name),
					err.Error(),
				))
			}
			signatures[name] = signature
		}
	}

	if diags.HasErrors() {
		return nil, diags
	}

	return signatures, nil
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

// marshalConvert returns a static function signature for the try function.
// We need this exception because the function implementation uses capsule
// types that we can't marshal.
func marshalConvert(convert function.Function) *FunctionSignature {
	return &FunctionSignature{
		Description: convert.Description(),
		ReturnType:  cty.DynamicPseudoType,
		Parameters: []*parameter{
			{
				Name:        convert.Params()[0].Name,
				Description: convert.Params()[0].Description,
				IsNullable:  convert.Params()[0].AllowNull,
				Type:        cty.DynamicPseudoType,
			},
			{
				Name:        convert.Params()[1].Name,
				Description: convert.Params()[1].Description,
				Type:        cty.DynamicPseudoType,
			},
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
