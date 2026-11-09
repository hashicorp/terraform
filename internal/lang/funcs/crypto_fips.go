//go:build fips

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var Md5Func = makeFipsDisallowedFunction(stringHashFunctionParams)

func MakeFileMd5Func(baseDir string, wrap ImplWrapper) function.Function {
	return makeFipsDisallowedFunction(fileHashFunctionParams)
}

var Sha1Func = makeFipsDisallowedFunction(stringHashFunctionParams)

func MakeFileSha1Func(baseDir string, wrap ImplWrapper) function.Function {
	return makeFipsDisallowedFunction(fileHashFunctionParams)
}

func makeFipsDisallowedFunction(params []function.Parameter) function.Function {
	err := FipsDisallowedErr{}
	return function.New(&function.Spec{
		Params: params,
		Type: func(args []cty.Value) (cty.Type, error) {
			return cty.DynamicPseudoType, err
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			return cty.DynamicVal, err
		},
	})
}

type FipsDisallowedErr struct{}

func (e FipsDisallowedErr) Error() string {
	return "this function is not allowed in FIPS 140"
}
