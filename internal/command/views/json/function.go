// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package json

import (
	"encoding/json"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// Function is a description of the JSON representation of the signature of
// a function callable from the Terraform language.
type Function struct {
	// Name is the leaf name of the function, without any namespace prefix.
	Name string `json:"name"`

	Params        []FunctionParam `json:"params"`
	VariadicParam *FunctionParam  `json:"variadic_param,omitempty"`

	// ReturnType is type constraint which is a static approximation of the
	// possibly-dynamic return type of the function.
	ReturnType json.RawMessage `json:"return_type"`

	Description     string `json:"description,omitempty"`
	DescriptionKind string `json:"description_kind,omitempty"`
}

// FunctionParam represents a single parameter to a function, as represented
// by type Function.
type FunctionParam struct {
	// Name is a name for the function which is used primarily for
	// documentation purposes, because function arguments are positional
	// and therefore don't appear directly in configuration source code.
	Name string `json:"name"`

	// Type is a type constraint which is a static approximation of the
	// possibly-dynamic type of the parameter. Particular functions may
	// have additional requirements that a type constraint alone cannot
	// represent.
	Type json.RawMessage `json:"type"`

	// Maybe some of the other fields in function.Parameter would be
	// interesting to describe here too, but we'll wait to see if there
	// is a use-case first.

	Description     string `json:"description,omitempty"`
	DescriptionKind string `json:"description_kind,omitempty"`
}

// DescribeFunction returns a description of the signature of the given cty
// function, as a pointer to this package's serializable type Function.
func DescribeFunction(name string, f function.Function) *Function {
	ret := &Function{
		Name: name,
	}

	params := f.Params()
	ret.Params = make([]FunctionParam, len(params))
	typeCheckArgs := make([]cty.Type, len(params), len(params)+1)
	for i, param := range params {
		ret.Params[i] = describeFunctionParam(&param)
		typeCheckArgs[i] = param.Type
	}
	if varParam := f.VarParam(); varParam != nil {
		descParam := describeFunctionParam(varParam)
		ret.VariadicParam = &descParam
		typeCheckArgs = append(typeCheckArgs, varParam.Type)
	}

	retType, err := f.ReturnType(typeCheckArgs)
	if err != nil {
		// Getting an error when type-checking with exactly the type constraints
		// the function called for is weird, so we'll just treat it as if it
		// has a dynamic return type instead, for our purposes here.
		// One reason this can happen is for a function which has a variadic
		// parameter but has logic inside it which considers it invalid to
		// specify exactly one argument for that parameter (since that's what
		// we did in typeCheckArgs as an approximation of a valid call above.)
		retType = cty.DynamicPseudoType
	}

	if raw, err := retType.MarshalJSON(); err != nil {
		// Again, we'll treat any errors as if the function is dynamically
		// typed because it would be weird to get here.
		ret.ReturnType = json.RawMessage(`"dynamic"`)
	} else {
		ret.ReturnType = json.RawMessage(raw)
	}

	// We don't currently have any sense of descriptions for functions and
	// their parameters, so we'll just leave those fields unpopulated for now.

	return ret
}

func describeFunctionParam(p *function.Parameter) FunctionParam {
	ret := FunctionParam{
		Name: p.Name,
	}

	if raw, err := p.Type.MarshalJSON(); err != nil {
		// We'll treat any errors as if the function is dynamically
		// typed because it would be weird to get here.
		ret.Type = json.RawMessage(`"dynamic"`)
	} else {
		ret.Type = json.RawMessage(raw)
	}

	// We don't currently have any sense of descriptions for functions and
	// their parameters, so we'll just leave those fields unpopulated for now.

	return ret
}
