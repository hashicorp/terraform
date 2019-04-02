package hcl2shim

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/hil/ast"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
)

// UnknownVariableValue is a sentinel value that can be used
// to denote that the value of a variable is unknown at this time.
// RawConfig uses this information to build up data about
// unknown keys.
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// ConfigValueFromHCL2Block is like ConfigValueFromHCL2 but it works only for
// known object values and uses the provided block schema to perform some
// additional normalization to better mimic the shape of value that the old
// HCL1/HIL-based codepaths would've produced.
//
// In particular, it discards the collections that we use to represent nested
// blocks (other than NestingSingle) if they are empty, which better mimics
// the HCL1 behavior because HCL1 had no knowledge of the schema and so didn't
// know that an unspecified block _could_ exist.
//
// The given object value must conform to the schema's implied type or this
// function will panic or produce incorrect results.
//
// This is primarily useful for the final transition from new-style values to
// terraform.ResourceConfig before calling to a legacy provider, since
// helper/schema (the old provider SDK) is particularly sensitive to these
// subtle differences within its validation code.
func ConfigValueFromHCL2Block(v cty.Value, schema *configschema.Block) map[string]interface{} {
	if v.IsNull() {
		return nil
	}
	if !v.IsKnown() {
		panic("ConfigValueFromHCL2Block used with unknown value")
	}
	if !v.Type().IsObjectType() {
		panic(fmt.Sprintf("ConfigValueFromHCL2Block used with non-object value %#v", v))
	}

	atys := v.Type().AttributeTypes()
	ret := make(map[string]interface{})

	for name := range schema.Attributes {
		if _, exists := atys[name]; !exists {
			continue
		}

		av := v.GetAttr(name)
		if av.IsNull() {
			// Skip nulls altogether, to better mimic how HCL1 would behave
			continue
		}
		ret[name] = ConfigValueFromHCL2(av)
	}

	for name, blockS := range schema.BlockTypes {
		if _, exists := atys[name]; !exists {
			continue
		}
		bv := v.GetAttr(name)
		if !bv.IsKnown() {
			ret[name] = UnknownVariableValue
			continue
		}
		if bv.IsNull() {
			continue
		}

		switch blockS.Nesting {

		case configschema.NestingSingle:
			ret[name] = ConfigValueFromHCL2Block(bv, &blockS.Block)

		case configschema.NestingList, configschema.NestingSet:
			l := bv.LengthInt()
			if l == 0 {
				// skip empty collections to better mimic how HCL1 would behave
				continue
			}

			elems := make([]interface{}, 0, l)
			for it := bv.ElementIterator(); it.Next(); {
				_, ev := it.Element()
				if !ev.IsKnown() {
					elems = append(elems, UnknownVariableValue)
					continue
				}
				elems = append(elems, ConfigValueFromHCL2Block(ev, &blockS.Block))
			}
			ret[name] = elems

		case configschema.NestingMap:
			if bv.LengthInt() == 0 {
				// skip empty collections to better mimic how HCL1 would behave
				continue
			}

			elems := make(map[string]interface{})
			for it := bv.ElementIterator(); it.Next(); {
				ek, ev := it.Element()
				if !ev.IsKnown() {
					elems[ek.AsString()] = UnknownVariableValue
					continue
				}
				elems[ek.AsString()] = ConfigValueFromHCL2Block(ev, &blockS.Block)
			}
			ret[name] = elems
		}
	}

	return ret
}

// ConfigValueFromHCL2 converts a value from HCL2 (really, from the cty dynamic
// types library that HCL2 uses) to a value type that matches what would've
// been produced from the HCL-based interpolator for an equivalent structure.
//
// This function will transform a cty null value into a Go nil value, which
// isn't a possible outcome of the HCL/HIL-based decoder and so callers may
// need to detect and reject any null values.
func ConfigValueFromHCL2(v cty.Value) interface{} {
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
			l = append(l, ConfigValueFromHCL2(ev))
		}
		return l
	}

	if v.Type().IsMapType() || v.Type().IsObjectType() {
		l := make(map[string]interface{})
		it := v.ElementIterator()
		for it.Next() {
			ek, ev := it.Element()
			cv := ConfigValueFromHCL2(ev)
			if cv != nil {
				l[ek.AsString()] = cv
			}
		}
		return l
	}

	// If we fall out here then we have some weird type that we haven't
	// accounted for. This should never happen unless the caller is using
	// capsule types, and we don't currently have any such types defined.
	panic(fmt.Errorf("can't convert %#v to config value", v))
}

// HCL2ValueFromConfigValue is the opposite of configValueFromHCL2: it takes
// a value as would be returned from the old interpolator and turns it into
// a cty.Value so it can be used within, for example, an HCL2 EvalContext.
func HCL2ValueFromConfigValue(v interface{}) cty.Value {
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
			vals[i] = HCL2ValueFromConfigValue(ev)
		}
		return cty.TupleVal(vals)
	case map[string]interface{}:
		vals := map[string]cty.Value{}
		for k, ev := range tv {
			vals[k] = HCL2ValueFromConfigValue(ev)
		}
		return cty.ObjectVal(vals)
	default:
		// HCL/HIL should never generate anything that isn't caught by
		// the above, so if we get here something has gone very wrong.
		panic(fmt.Errorf("can't convert %#v to cty.Value", v))
	}
}

func HILVariableFromHCL2Value(v cty.Value) ast.Variable {
	if v.IsNull() {
		// Caller should guarantee/check this before calling
		panic("Null values cannot be represented in HIL")
	}
	if !v.IsKnown() {
		return ast.Variable{
			Type:  ast.TypeUnknown,
			Value: UnknownVariableValue,
		}
	}

	switch v.Type() {
	case cty.Bool:
		return ast.Variable{
			Type:  ast.TypeBool,
			Value: v.True(),
		}
	case cty.Number:
		v := ConfigValueFromHCL2(v)
		switch tv := v.(type) {
		case int:
			return ast.Variable{
				Type:  ast.TypeInt,
				Value: tv,
			}
		case float64:
			return ast.Variable{
				Type:  ast.TypeFloat,
				Value: tv,
			}
		default:
			// should never happen
			panic("invalid return value for configValueFromHCL2")
		}
	case cty.String:
		return ast.Variable{
			Type:  ast.TypeString,
			Value: v.AsString(),
		}
	}

	if v.Type().IsListType() || v.Type().IsSetType() || v.Type().IsTupleType() {
		l := make([]ast.Variable, 0, v.LengthInt())
		it := v.ElementIterator()
		for it.Next() {
			_, ev := it.Element()
			l = append(l, HILVariableFromHCL2Value(ev))
		}
		// If we were given a tuple then this could actually produce an invalid
		// list with non-homogenous types, which we expect to be caught inside
		// HIL just like a user-supplied non-homogenous list would be.
		return ast.Variable{
			Type:  ast.TypeList,
			Value: l,
		}
	}

	if v.Type().IsMapType() || v.Type().IsObjectType() {
		l := make(map[string]ast.Variable)
		it := v.ElementIterator()
		for it.Next() {
			ek, ev := it.Element()
			l[ek.AsString()] = HILVariableFromHCL2Value(ev)
		}
		// If we were given an object then this could actually produce an invalid
		// map with non-homogenous types, which we expect to be caught inside
		// HIL just like a user-supplied non-homogenous map would be.
		return ast.Variable{
			Type:  ast.TypeMap,
			Value: l,
		}
	}

	// If we fall out here then we have some weird type that we haven't
	// accounted for. This should never happen unless the caller is using
	// capsule types, and we don't currently have any such types defined.
	panic(fmt.Errorf("can't convert %#v to HIL variable", v))
}

func HCL2ValueFromHILVariable(v ast.Variable) cty.Value {
	switch v.Type {
	case ast.TypeList:
		vals := make([]cty.Value, len(v.Value.([]ast.Variable)))
		for i, ev := range v.Value.([]ast.Variable) {
			vals[i] = HCL2ValueFromHILVariable(ev)
		}
		return cty.TupleVal(vals)
	case ast.TypeMap:
		vals := make(map[string]cty.Value, len(v.Value.(map[string]ast.Variable)))
		for k, ev := range v.Value.(map[string]ast.Variable) {
			vals[k] = HCL2ValueFromHILVariable(ev)
		}
		return cty.ObjectVal(vals)
	default:
		return HCL2ValueFromConfigValue(v.Value)
	}
}

func HCL2TypeForHILType(hilType ast.Type) cty.Type {
	switch hilType {
	case ast.TypeAny:
		return cty.DynamicPseudoType
	case ast.TypeUnknown:
		return cty.DynamicPseudoType
	case ast.TypeBool:
		return cty.Bool
	case ast.TypeInt:
		return cty.Number
	case ast.TypeFloat:
		return cty.Number
	case ast.TypeString:
		return cty.String
	case ast.TypeList:
		return cty.List(cty.DynamicPseudoType)
	case ast.TypeMap:
		return cty.Map(cty.DynamicPseudoType)
	default:
		return cty.NilType // equilvalent to ast.TypeInvalid
	}
}
