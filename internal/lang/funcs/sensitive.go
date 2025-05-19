// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// SensitiveFunc returns a value identical to its argument except that
// Terraform will consider it to be sensitive.
var SensitiveFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowUnknown:     true,
			AllowNull:        true,
			AllowMarked:      true,
			AllowDynamicType: true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// This function only affects the value's marks, so the result
		// type is always the same as the argument type.
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		return args[0].Mark(marks.Sensitive), nil
	},
})

// NonsensitiveFunc takes a sensitive value and returns the same value without
// the sensitive marking, effectively exposing the value.
var NonsensitiveFunc = function.New(&function.Spec{
	Params: []function.Parameter{
		{
			Name:             "value",
			Type:             cty.DynamicPseudoType,
			AllowUnknown:     true,
			AllowNull:        true,
			AllowMarked:      true,
			AllowDynamicType: true,
		},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		// This function only affects the value's marks, so the result
		// type is always the same as the argument type.
		return args[0].Type(), nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (ret cty.Value, err error) {
		v, m := args[0].Unmark()
		delete(m, marks.Sensitive) // remove the sensitive marking
		return v.WithMarks(m), nil
	},
})

var IssensitiveFunc = function.New(&function.Spec{
	Params: []function.Parameter{{
		Name:             "value",
		Type:             cty.DynamicPseudoType,
		AllowUnknown:     true,
		AllowNull:        true,
		AllowMarked:      true,
		AllowDynamicType: true,
	}},
	Type: func(args []cty.Value) (cty.Type, error) {
		return cty.Bool, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		switch v := args[0]; {
		case v.HasMark(marks.Sensitive):
			return cty.True, nil
		case !v.IsKnown():
			return cty.UnknownVal(cty.Bool), nil
		default:
			return cty.False, nil
		}
	},
})

func Sensitive(v cty.Value) (cty.Value, error) {
	return SensitiveFunc.Call([]cty.Value{v})
}

func Nonsensitive(v cty.Value) (cty.Value, error) {
	return NonsensitiveFunc.Call([]cty.Value{v})
}

func Issensitive(v cty.Value) (cty.Value, error) {
	return IssensitiveFunc.Call([]cty.Value{v})
}
