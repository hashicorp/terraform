package config

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

// ---------------------------------------------------------------------------
// This file contains some helper functions that are used to shim between
// HCL2 concepts and HCL/HIL concepts, to help us mostly preserve the existing
// public API that was built around HCL/HIL-oriented approaches.
// ---------------------------------------------------------------------------

// configValueFromHCL2 converts a value from HCL2 (really, from the cty dynamic
// types library that HCL2 uses) to a value type that matches what would've
// been produced from the HCL-based interpolator for an equivalent structure.
//
// This function will transform a cty null value into a Go nil value, which
// isn't a possible outcome of the HCL/HIL-based decoder and so callers may
// need to detect and reject any null values.
func configValueFromHCL2(v cty.Value) interface{} {
	if !v.IsKnown() {
		return UnknownVariableValue
	}
	if v.IsNull() {
		return nil
	}

	switch v.Type() {
	case cty.Bool:
		return v.True() // like HCL.BOOL
	case cty.String:
		return v.AsString() // like HCL token.STRING or token.HEREDOC
	case cty.Number:
		// We can't match HCL _exactly_ here because it distinguishes between
		// int and float values, but we'll get as close as we can by using
		// an int if the number is exactly representable, and a float if not.
		// The conversion to float will force precision to that of a float64,
		// which is potentially losing information from the specific number
		// given, but no worse than what HCL would've done in its own conversion
		// to float.

		f := v.AsBigFloat()
		if i, acc := f.Int64(); acc == big.Exact {
			// if we're on a 32-bit system and the number is too big for 32-bit
			// int then we'll fall through here and use a float64.
			const MaxInt = int(^uint(0) >> 1)
			const MinInt = -MaxInt - 1
			if i <= int64(MaxInt) && i >= int64(MinInt) {
				return int(i) // Like HCL token.NUMBER
			}
		}

		f64, _ := f.Float64()
		return f64 // like HCL token.FLOAT
	}

	if v.Type().IsListType() || v.Type().IsSetType() || v.Type().IsTupleType() {
		l := make([]interface{}, 0, v.LengthInt())
		it := v.ElementIterator()
		for it.Next() {
			_, ev := it.Element()
			l = append(l, configValueFromHCL2(ev))
		}
		return l
	}

	if v.Type().IsMapType() || v.Type().IsObjectType() {
		l := make(map[string]interface{})
		it := v.ElementIterator()
		for it.Next() {
			ek, ev := it.Element()
			l[ek.AsString()] = configValueFromHCL2(ev)
		}
		return l
	}

	// If we fall out here then we have some weird type that we haven't
	// accounted for. This should never happen unless the caller is using
	// capsule types, and we don't currently have any such types defined.
	panic(fmt.Errorf("can't convert %#v to config value", v))
}

// hcl2ValueFromConfigValue is the opposite of configValueFromHCL2: it takes
// a value as would be returned from the old interpolator and turns it into
// a cty.Value so it can be used within, for example, an HCL2 EvalContext.
func hcl2ValueFromConfigValue(v interface{}) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	if v == UnknownVariableValue {
		return cty.DynamicVal
	}

	switch tv := v.(type) {
	case bool:
		return cty.BoolVal(tv)
	case string:
		return cty.StringVal(tv)
	case int:
		return cty.NumberIntVal(int64(tv))
	case float64:
		return cty.NumberFloatVal(tv)
	case []interface{}:
		vals := make([]cty.Value, len(tv))
		for i, ev := range tv {
			vals[i] = hcl2ValueFromConfigValue(ev)
		}
		return cty.TupleVal(vals)
	case map[string]interface{}:
		vals := map[string]cty.Value{}
		for k, ev := range tv {
			vals[k] = hcl2ValueFromConfigValue(ev)
		}
		return cty.ObjectVal(vals)
	default:
		// HCL/HIL should never generate anything that isn't caught by
		// the above, so if we get here something has gone very wrong.
		panic(fmt.Errorf("can't convert %#v to cty.Value", v))
	}
}

func hcl2InterpolationFuncs() map[string]function.Function {
	hcl2Funcs := map[string]function.Function{}

	for name, hilFunc := range Funcs() {
		hcl2Funcs[name] = hcl2InterpolationFuncShim(&hilFunc)
	}

	return hcl2Funcs
}

func hcl2InterpolationFuncShim(hilFunc *ast.Function) function.Function {
	spec := &function.Spec{}
	spec.Impl = func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		hilArgs := make([]interface{}, len(args))
		for i, arg := range args {
			rv := configValueFromHCL2(arg)
			hilV, err := hil.InterfaceToVariable(rv)
			if err != nil {
				return cty.DynamicVal, err
			}
			// HIL functions actually expect to have the outermost variable
			// "peeled" but any nested values (in lists or maps) will
			// still have their ast.Variable wrapping.
			hilArgs[i] = hilV.Value
		}

		hilResult, err := hilFunc.Callback(hilArgs)

		// Just as on the way in, we get back a partially-peeled ast.Variable
		// which we need to re-wrap in order to convert it back into what
		// we're calling a "config value".

		rr, err := hil.VariableToInterface(ast.Variable{
			Type:  hilFunc.ReturnType,
			Value: hilResult,
		})
		if err != nil {
			return cty.DynamicVal, err
		}

		return hcl2ValueFromConfigValue(rr), nil
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

// hcl2SingleAttrBody is a weird implementation of hcl2.Body that acts as if
// it has a single attribute whose value is the given expression.
//
// This is used to shim Resource.RawCount and Output.RawConfig to behave
// more like they do in the old HCL loader.
type hcl2SingleAttrBody struct {
	Name string
	Expr hcl2.Expression
}

var _ hcl2.Body = hcl2SingleAttrBody{}

func (b hcl2SingleAttrBody) Content(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Diagnostics) {
	content, all, diags := b.content(schema)
	if !all {
		// This should never happen because this body implementation should only
		// be used by code that is aware that it's using a single-attr body.
		diags = append(diags, &hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid attribute",
			Detail:   fmt.Sprintf("The correct attribute name is %q.", b.Name),
			Subject:  b.Expr.Range().Ptr(),
		})
	}
	return content, diags
}

func (b hcl2SingleAttrBody) PartialContent(schema *hcl2.BodySchema) (*hcl2.BodyContent, hcl2.Body, hcl2.Diagnostics) {
	content, all, diags := b.content(schema)
	var remain hcl2.Body
	if all {
		// If the request matched the one attribute we represent, then the
		// remaining body is empty.
		remain = hcl2.EmptyBody()
	} else {
		remain = b
	}
	return content, remain, diags
}

func (b hcl2SingleAttrBody) content(schema *hcl2.BodySchema) (*hcl2.BodyContent, bool, hcl2.Diagnostics) {
	ret := &hcl2.BodyContent{}
	all := false
	var diags hcl2.Diagnostics

	for _, attrS := range schema.Attributes {
		if attrS.Name == b.Name {
			attrs, _ := b.JustAttributes()
			ret.Attributes = attrs
			all = true
		} else if attrS.Required {
			diags = append(diags, &hcl2.Diagnostic{
				Severity: hcl2.DiagError,
				Summary:  "Missing attribute",
				Detail:   fmt.Sprintf("The attribute %q is required.", attrS.Name),
				Subject:  b.Expr.Range().Ptr(),
			})
		}
	}

	return ret, all, diags
}

func (b hcl2SingleAttrBody) JustAttributes() (hcl2.Attributes, hcl2.Diagnostics) {
	return hcl2.Attributes{
		b.Name: {
			Expr:      b.Expr,
			Name:      b.Name,
			NameRange: b.Expr.Range(),
			Range:     b.Expr.Range(),
		},
	}, nil
}

func (b hcl2SingleAttrBody) MissingItemRange() hcl2.Range {
	return b.Expr.Range()
}
