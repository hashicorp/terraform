package config

import (
	"fmt"

	"github.com/zclconf/go-cty/cty/function/stdlib"

	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/configs/hcl2shim"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// ---------------------------------------------------------------------------
// This file contains some helper functions that are used to shim between
// HCL2 concepts and HCL/HIL concepts, to help us mostly preserve the existing
// public API that was built around HCL/HIL-oriented approaches.
// ---------------------------------------------------------------------------

func hcl2InterpolationFuncs() map[string]function.Function {
	hcl2Funcs := map[string]function.Function{}

	for name, hilFunc := range Funcs() {
		hcl2Funcs[name] = hcl2InterpolationFuncShim(hilFunc)
	}

	// Some functions in the old world are dealt with inside langEvalConfig
	// due to their legacy reliance on direct access to the symbol table.
	// Since 0.7 they don't actually need it anymore and just ignore it,
	// so we're cheating a bit here and exploiting that detail by passing nil.
	hcl2Funcs["lookup"] = hcl2InterpolationFuncShim(interpolationFuncLookup(nil))
	hcl2Funcs["keys"] = hcl2InterpolationFuncShim(interpolationFuncKeys(nil))
	hcl2Funcs["values"] = hcl2InterpolationFuncShim(interpolationFuncValues(nil))

	// As a bonus, we'll provide the JSON-handling functions from the cty
	// function library since its "jsonencode" is more complete (doesn't force
	// weird type conversions) and HIL's type system can't represent
	// "jsondecode" at all. The result of jsondecode will eventually be forced
	// to conform to the HIL type system on exit into the rest of Terraform due
	// to our shimming right now, but it should be usable for decoding _within_
	// an expression.
	hcl2Funcs["jsonencode"] = stdlib.JSONEncodeFunc
	hcl2Funcs["jsondecode"] = stdlib.JSONDecodeFunc

	return hcl2Funcs
}

func hcl2InterpolationFuncShim(hilFunc ast.Function) function.Function {
	spec := &function.Spec{}

	for i, hilArgType := range hilFunc.ArgTypes {
		spec.Params = append(spec.Params, function.Parameter{
			Type: hcl2shim.HCL2TypeForHILType(hilArgType),
			Name: fmt.Sprintf("arg%d", i+1), // HIL args don't have names, so we'll fudge it
		})
	}

	if hilFunc.Variadic {
		spec.VarParam = &function.Parameter{
			Type: hcl2shim.HCL2TypeForHILType(hilFunc.VariadicType),
			Name: "varargs", // HIL args don't have names, so we'll fudge it
		}
	}

	spec.Type = func(args []cty.Value) (cty.Type, error) {
		return hcl2shim.HCL2TypeForHILType(hilFunc.ReturnType), nil
	}
	spec.Impl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		hilArgs := make([]interface{}, len(args))
		for i, arg := range args {
			hilV := hcl2shim.HILVariableFromHCL2Value(arg)

			// Although the cty function system does automatic type conversions
			// to match the argument types, cty doesn't distinguish int and
			// float and so we may need to adjust here to ensure that the
			// wrapped function gets exactly the Go type it was expecting.
			var wantType ast.Type
			if i < len(hilFunc.ArgTypes) {
				wantType = hilFunc.ArgTypes[i]
			} else {
				wantType = hilFunc.VariadicType
			}
			switch {
			case hilV.Type == ast.TypeInt && wantType == ast.TypeFloat:
				hilV.Type = wantType
				hilV.Value = float64(hilV.Value.(int))
			case hilV.Type == ast.TypeFloat && wantType == ast.TypeInt:
				hilV.Type = wantType
				hilV.Value = int(hilV.Value.(float64))
			}

			// HIL functions actually expect to have the outermost variable
			// "peeled" but any nested values (in lists or maps) will
			// still have their ast.Variable wrapping.
			hilArgs[i] = hilV.Value
		}

		hilResult, err := hilFunc.Callback(hilArgs)
		if err != nil {
			return cty.DynamicVal, err
		}

		// Just as on the way in, we get back a partially-peeled ast.Variable
		// which we need to re-wrap in order to convert it back into what
		// we're calling a "config value".
		rv := hcl2shim.HCL2ValueFromHILVariable(ast.Variable{
			Type:  hilFunc.ReturnType,
			Value: hilResult,
		})

		return convert.Convert(rv, retType) // if result is unknown we'll force the correct type here
	}
	return function.New(spec)
}

func hcl2EvalWithUnknownVars(expr hcl2.Expression) (cty.Value, hcl2.Diagnostics) {
	trs := expr.Variables()
	vars := map[string]cty.Value{}
	val := cty.DynamicVal

	for _, tr := range trs {
		name := tr.RootName()
		vars[name] = val
	}

	ctx := &hcl2.EvalContext{
		Variables: vars,
		Functions: hcl2InterpolationFuncs(),
	}
	return expr.Value(ctx)
}
