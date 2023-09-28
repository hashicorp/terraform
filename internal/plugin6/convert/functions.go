package convert

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfplugin6"
)

func FunctionDeclsFromProto(protoFuncs map[string]*tfplugin6.Function) (map[string]providers.FunctionDecl, error) {
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

func FunctionDeclFromProto(protoFunc *tfplugin6.Function) (providers.FunctionDecl, error) {
	var ret providers.FunctionDecl

	ret.Description = protoFunc.Description
	ret.DescriptionKind = schemaStringKind(protoFunc.DescriptionKind)

	if err := json.Unmarshal(protoFunc.ReturnType, &ret.ReturnType); err != nil {
		return ret, fmt.Errorf("invalid return type constraint: %s", err)
	}

	if len(protoFunc.Params) != 0 {
		ret.Parameters = make([]providers.FunctionParam, len(protoFunc.Params))
		for i, protoParam := range protoFunc.Params {
			param, err := functionParamFromProto(protoParam)
			if err != nil {
				return ret, fmt.Errorf("invalid parameter %d (%q): %s", i, protoParam.Name, err)
			}
			ret.Parameters[i] = param
		}
	}
	if protoFunc.VariadicParam != nil {
		param, err := functionParamFromProto(protoFunc.VariadicParam)
		if err != nil {
			return ret, fmt.Errorf("invalid variadic parameter (%q): %s", protoFunc.VariadicParam.Name, err)
		}
		ret.VariadicParameter = &param
	}

	return ret, nil
}

func functionParamFromProto(protoParam *tfplugin6.Function_Parameter) (providers.FunctionParam, error) {
	var ret providers.FunctionParam
	ret.Name = protoParam.Name
	ret.Description = protoParam.Description
	ret.DescriptionKind = schemaStringKind(protoParam.DescriptionKind)
	ret.Nullable = protoParam.Nullable
	ret.AllowUnknownValues = protoParam.AllowUnknownValues
	if err := json.Unmarshal(protoParam.Type, &ret.Type); err != nil {
		return ret, fmt.Errorf("invalid type constraint: %s", err)
	}
	return ret, nil
}
