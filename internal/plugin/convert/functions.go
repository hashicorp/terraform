// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin5"
)

func FunctionDeclsFromProto(protoFuncs map[string]*tfplugin5.Function) (map[string]providers.FunctionDecl, error) {
	if len(protoFuncs) == 0 {
		return nil, nil
	}

	ret := make(map[string]providers.FunctionDecl, len(protoFuncs))
	for name, protoFunc := range protoFuncs {
		decl, err := FunctionDeclFromProto(protoFunc)
		if err != nil {
			return nil, fmt.Errorf("invalid declaration for function %q: %s", name, err)
		}
		ret[name] = decl
	}
	return ret, nil
}

func FunctionDeclFromProto(protoFunc *tfplugin5.Function) (providers.FunctionDecl, error) {
	var ret providers.FunctionDecl

	ret.Description = protoFunc.Description
	ret.DescriptionKind = schemaStringKind(protoFunc.DescriptionKind)
	ret.Summary = protoFunc.Summary
	ret.DeprecationMessage = protoFunc.DeprecationMessage

	if err := json.Unmarshal(protoFunc.Return.Type, &ret.ReturnType); err != nil {
		return ret, fmt.Errorf("invalid return type constraint: %s", err)
	}

	if len(protoFunc.Parameters) != 0 {
		ret.Parameters = make([]providers.FunctionParam, len(protoFunc.Parameters))
		for i, protoParam := range protoFunc.Parameters {
			param, err := functionParamFromProto(protoParam)
			if err != nil {
				return ret, fmt.Errorf("invalid parameter %d (%q): %s", i, protoParam.Name, err)
			}
			ret.Parameters[i] = param
		}
	}
	if protoFunc.VariadicParameter != nil {
		param, err := functionParamFromProto(protoFunc.VariadicParameter)
		if err != nil {
			return ret, fmt.Errorf("invalid variadic parameter (%q): %s", protoFunc.VariadicParameter.Name, err)
		}
		ret.VariadicParameter = &param
	}

	return ret, nil
}

func functionParamFromProto(protoParam *tfplugin5.Function_Parameter) (providers.FunctionParam, error) {
	var ret providers.FunctionParam
	ret.Name = protoParam.Name
	ret.Description = protoParam.Description
	ret.DescriptionKind = schemaStringKind(protoParam.DescriptionKind)
	ret.AllowNullValue = protoParam.AllowNullValue
	ret.AllowUnknownValues = protoParam.AllowUnknownValues
	if err := json.Unmarshal(protoParam.Type, &ret.Type); err != nil {
		return ret, fmt.Errorf("invalid type constraint: %s", err)
	}
	return ret, nil
}

func FunctionDeclsToProto(fns map[string]providers.FunctionDecl) (map[string]*tfplugin5.Function, error) {
	if len(fns) == 0 {
		return nil, nil
	}

	ret := make(map[string]*tfplugin5.Function, len(fns))
	for name, fn := range fns {
		decl, err := FunctionDeclToProto(fn)
		if err != nil {
			return nil, fmt.Errorf("invalid declaration for function %q: %s", name, err)
		}
		ret[name] = decl
	}
	return ret, nil
}

func FunctionDeclToProto(fn providers.FunctionDecl) (*tfplugin5.Function, error) {
	ret := &tfplugin5.Function{
		Return: &tfplugin5.Function_Return{},
	}

	ret.Description = fn.Description
	ret.DescriptionKind = protoStringKind(fn.DescriptionKind)

	retTy, err := json.Marshal(fn.ReturnType)
	if err != nil {
		return ret, fmt.Errorf("invalid return type constraint: %s", err)
	}
	ret.Return.Type = retTy

	if len(fn.Parameters) != 0 {
		ret.Parameters = make([]*tfplugin5.Function_Parameter, len(fn.Parameters))
		for i, fnParam := range fn.Parameters {
			protoParam, err := functionParamToProto(fnParam)
			if err != nil {
				return ret, fmt.Errorf("invalid parameter %d (%q): %s", i, fnParam.Name, err)
			}
			ret.Parameters[i] = protoParam
		}
	}
	if fn.VariadicParameter != nil {
		param, err := functionParamToProto(*fn.VariadicParameter)
		if err != nil {
			return ret, fmt.Errorf("invalid variadic parameter (%q): %s", fn.VariadicParameter.Name, err)
		}
		ret.VariadicParameter = param
	}

	return ret, nil
}

func functionParamToProto(param providers.FunctionParam) (*tfplugin5.Function_Parameter, error) {
	ret := &tfplugin5.Function_Parameter{}
	ret.Name = param.Name
	ret.Description = param.Description
	ret.DescriptionKind = protoStringKind(param.DescriptionKind)
	ret.AllowNullValue = param.AllowNullValue
	ret.AllowUnknownValues = param.AllowUnknownValues
	ty, err := json.Marshal(param.Type)
	if err != nil {
		return ret, fmt.Errorf("invalid type constraint: %s", err)
	}
	ret.Type = ty
	return ret, nil
}
