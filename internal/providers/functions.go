// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang"
)

type FunctionDecl struct {
	Parameters        []FunctionParam
	VariadicParameter *FunctionParam
	ReturnType        cty.Type

	Description        string
	DescriptionKind    configschema.StringKind
	Summary            string
	DeprecationMessage string
}

type FunctionParam struct {
	Name string // Only for documentation and UI, because arguments are positional
	Type cty.Type

	AllowNullValue     bool
	AllowUnknownValues bool

	Description     string
	DescriptionKind configschema.StringKind
}

// BuildFunction takes a factory function which will return an unconfigured
// instance of the provider this declaration belongs to and returns a
// cty function that is ready to be called against that provider.
//
// The given name must be the name under which the provider originally
// registered this declaration, or the returned function will try to use an
// invalid name, leading to errors or undefined behavior.
//
// If the given factory returns an instance of any provider other than the
// one the declaration belongs to, or returns a _configured_ instance of
// the provider rather than an unconfigured one, the behavior of the returned
// function is undefined.
//
// Although not functionally required, callers should ideally pass a factory
// function that either retrieves already-running plugins or memoizes the
// plugins it returns so that many calls to functions in the same provider
// will not incur a repeated startup cost.
//
// The resTable argument is a shared instance of *FunctionResults, used to
// check the result values from each function call.
func (d FunctionDecl) BuildFunction(providerAddr addrs.Provider, name string, resTable *lang.FunctionResults, factory func() (Interface, error)) function.Function {

	var params []function.Parameter
	var varParam *function.Parameter
	if len(d.Parameters) > 0 {
		params = make([]function.Parameter, len(d.Parameters))
		for i, paramDecl := range d.Parameters {
			params[i] = paramDecl.ctyParameter()
		}
	}
	if d.VariadicParameter != nil {
		cp := d.VariadicParameter.ctyParameter()
		varParam = &cp
	}

	return function.New(&function.Spec{
		Type:     function.StaticReturnType(d.ReturnType),
		Params:   params,
		VarParam: varParam,
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			for i, arg := range args {
				var param function.Parameter
				if i < len(params) {
					param = params[i]
				} else {
					param = *varParam
				}

				// We promise provider developers that we won't pass them even
				// _nested_ unknown values unless they opt in to dealing with
				// them.
				if !param.AllowUnknown {
					if !arg.IsWhollyKnown() {
						return cty.UnknownVal(retType), nil
					}
				}

				// We also ensure that null values are never passed where they
				// are not expected.
				if !param.AllowNull {
					if arg.IsNull() {
						return cty.UnknownVal(retType), fmt.Errorf("argument %q cannot be null", param.Name)
					}
				}
			}

			provider, err := factory()
			if err != nil {
				return cty.UnknownVal(retType), fmt.Errorf("failed to launch provider plugin: %s", err)
			}

			resp := provider.CallFunction(CallFunctionRequest{
				FunctionName: name,
				Arguments:    args,
			})
			if resp.Err != nil {
				return cty.UnknownVal(retType), resp.Err
			}

			if resp.Result == cty.NilVal {
				return cty.UnknownVal(retType), fmt.Errorf("provider returned no result and no errors")
			}

			if resTable != nil {
				err = resTable.CheckPriorProvider(providerAddr, name, args, resp.Result)
				if err != nil {
					return cty.UnknownVal(retType), err
				}
			}

			return resp.Result, nil
		},
	})
}

func (p *FunctionParam) ctyParameter() function.Parameter {
	return function.Parameter{
		Name:      p.Name,
		Type:      p.Type,
		AllowNull: p.AllowNullValue,

		// While the function may not allow DynamicVal, a `null` literal is
		// also dynamically typed. If the parameter is dynamically typed, then
		// we must allow this for `null` to pass through.
		AllowDynamicType: p.Type == cty.DynamicPseudoType,

		// NOTE: Setting this is not a sufficient implementation of
		// FunctionParam.AllowUnknownValues, because cty's function
		// system only blocks passing in a top-level unknown, but
		// our provider-contributed functions API promises to only
		// pass wholly-known values unless AllowUnknownValues is true.
		// The function implementation itself must also check this.
		AllowUnknown: p.AllowUnknownValues,
	}
}
