// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonfunction

import (
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

func getReturnType(f function.Function) (cty.Type, error) {
	args := make([]cty.Type, 0)
	for _, param := range f.Params() {
		args = append(args, param.Type)
	}
	if f.VarParam() != nil {
		args = append(args, f.VarParam().Type)
	}

	return f.ReturnType(args)
}
