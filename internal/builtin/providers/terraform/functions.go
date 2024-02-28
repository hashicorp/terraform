// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var functions = map[string]func([]cty.Value) (cty.Value, error){
	"tfvarsencode": tfvarsencodeFunc,
	"tfvarsdecode": tfvarsdecodeFunc,
	"exprencode":   exprencodeFunc,
}

func tfvarsencodeFunc(args []cty.Value) (cty.Value, error) {
	// These error checks should not be hit in practice because the language
	// runtime should check them before calling, so this is just for robustness
	// and completeness.
	if len(args) > 1 {
		return cty.NilVal, function.NewArgErrorf(1, "too many arguments; only one expected")
	}
	if len(args) == 0 {
		return cty.NilVal, fmt.Errorf("exactly one argument is required")
	}

	v := args[0]
	ty := v.Type()

	if v.IsNull() {
		// Our functions schema does not say we allow null values, so we should
		// not get to this error message if the caller respects the schema.
		return cty.NilVal, function.NewArgErrorf(1, "cannot encode a null value in tfvars syntax")
	}
	if !v.IsWhollyKnown() {
		return cty.UnknownVal(cty.String).RefineNotNull(), nil
	}

	var keys []string
	switch {
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		keys = make([]string, 0, len(atys))
		for key := range atys {
			keys = append(keys, key)
		}
	case ty.IsMapType():
		keys = make([]string, 0, v.LengthInt())
		for it := v.ElementIterator(); it.Next(); {
			k, _ := it.Element()
			keys = append(keys, k.AsString())
		}
	default:
		return cty.NilVal, function.NewArgErrorf(1, "invalid value to encode: must be an object whose attribute names will become the encoded variable names")
	}
	sort.Strings(keys)

	f := hclwrite.NewEmptyFile()
	body := f.Body()
	for _, key := range keys {
		if !hclsyntax.ValidIdentifier(key) {
			// We can only encode valid identifiers as tfvars keys, since
			// the HCL argument grammar requires them to be identifiers.
			return cty.NilVal, function.NewArgErrorf(1, "invalid variable name %q: must be a valid identifier, per Terraform's rules for input variable declarations", key)
		}

		// This index should not fail because we know that "key" is a valid
		// index from the logic above.
		v, _ := hcl.Index(v, cty.StringVal(key), nil)
		body.SetAttributeValue(key, v)
	}

	result := f.Bytes()
	return cty.StringVal(string(result)), nil
}

func tfvarsdecodeFunc(args []cty.Value) (cty.Value, error) {
	// These error checks should not be hit in practice because the language
	// runtime should check them before calling, so this is just for robustness
	// and completeness.
	if len(args) > 1 {
		return cty.NilVal, function.NewArgErrorf(1, "too many arguments; only one expected")
	}
	if len(args) == 0 {
		return cty.NilVal, fmt.Errorf("exactly one argument is required")
	}
	if args[0].Type() != cty.String {
		return cty.NilVal, fmt.Errorf("argument must be a string")
	}
	if args[0].IsNull() {
		return cty.NilVal, fmt.Errorf("cannot decode tfvars from a null value")
	}
	if !args[0].IsKnown() {
		// If our input isn't known then we can't even predict the result
		// type, since it will be an object type decided based on which
		// arguments and values we find in the string.
		return cty.DynamicVal, nil
	}

	// If we get here then we know that:
	// - there's exactly one element in args
	// - it's a string
	// - it is known and non-null
	// So therefore the following is guaranteed to succeed.
	src := []byte(args[0].AsString())

	// As usual when we wrap HCL stuff up in functions, we end up needing to
	// stuff HCL diagnostics into plain string error messages. This produces
	// a non-ideal result but is still better than hiding the HCL-provided
	// diagnosis altogether.
	f, hclDiags := hclsyntax.ParseConfig(src, "<tfvarsdecode argument>", hcl.InitialPos)
	if hclDiags.HasErrors() {
		return cty.NilVal, fmt.Errorf("invalid tfvars syntax: %s", hclDiags.Error())
	}
	attrs, hclDiags := f.Body.JustAttributes()
	if hclDiags.HasErrors() {
		return cty.NilVal, fmt.Errorf("invalid tfvars content: %s", hclDiags.Error())
	}
	retAttrs := make(map[string]cty.Value, len(attrs))
	for name, attr := range attrs {
		// Evaluating the expression with no EvalContext achieves the same
		// interpretation as Terraform CLI makes of .tfvars files, rejecting
		// any function calls or references to symbols.
		v, hclDiags := attr.Expr.Value(nil)
		if hclDiags.HasErrors() {
			return cty.NilVal, fmt.Errorf("invalid expression for variable %q: %s", name, hclDiags.Error())
		}
		retAttrs[name] = v
	}

	return cty.ObjectVal(retAttrs), nil
}

func exprencodeFunc(args []cty.Value) (cty.Value, error) {
	// These error checks should not be hit in practice because the language
	// runtime should check them before calling, so this is just for robustness
	// and completeness.
	if len(args) > 1 {
		return cty.NilVal, function.NewArgErrorf(1, "too many arguments; only one expected")
	}
	if len(args) == 0 {
		return cty.NilVal, fmt.Errorf("exactly one argument is required")
	}

	v := args[0]
	if !v.IsWhollyKnown() {
		ret := cty.UnknownVal(cty.String).RefineNotNull()
		// For some types we can refine further due to the HCL grammar,
		// as long as w eknow the value isn't null.
		if !v.Range().CouldBeNull() {
			ty := v.Type()
			switch {
			case ty.IsObjectType() || ty.IsMapType():
				ret = ret.Refine().StringPrefixFull("{").NewValue()
			case ty.IsTupleType() || ty.IsListType() || ty.IsSetType():
				ret = ret.Refine().StringPrefixFull("[").NewValue()
			case ty == cty.String:
				ret = ret.Refine().StringPrefixFull(`"`).NewValue()
			}
		}
		return ret, nil
	}

	// This bytes.TrimSpace is to ensure that future changes to HCL, that
	// might for some reason add extra spaces before the expression (!)
	// can't invalidate our unknown value prefix refinements above.
	src := bytes.TrimSpace(hclwrite.TokensForValue(v).Bytes())
	return cty.StringVal(string(src)), nil
}
