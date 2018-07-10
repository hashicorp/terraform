package hcl2shim

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zclconf/go-cty/cty/convert"

	"github.com/zclconf/go-cty/cty"
)

// FlatmapValueFromHCL2 converts a value from HCL2 (really, from the cty dynamic
// types library that HCL2 uses) to a map compatible with what would be
// produced by the "flatmap" package.
//
// The type of the given value informs the structure of the resulting map.
// The value must be of an object type or this function will panic.
//
// Flatmap values can only represent maps when they are of primitive types,
// so the given value must not have any maps of complex types or the result
// is undefined.
func FlatmapValueFromHCL2(v cty.Value) map[string]string {
	if !v.Type().IsObjectType() {
		panic(fmt.Sprintf("HCL2ValueFromFlatmap called on %#v", v.Type()))
	}

	m := make(map[string]string)
	flatmapValueFromHCL2Map(m, "", v)
	return m
}

func flatmapValueFromHCL2Value(m map[string]string, key string, val cty.Value) {
	ty := val.Type()
	switch {
	case ty.IsPrimitiveType():
		flatmapValueFromHCL2Primitive(m, key, val)
	case ty.IsObjectType() || ty.IsMapType():
		flatmapValueFromHCL2Map(m, key+".", val)
	case ty.IsTupleType() || ty.IsListType() || ty.IsSetType():
		flatmapValueFromHCL2Seq(m, key+".", val)
	default:
		panic(fmt.Sprintf("cannot encode %s to flatmap", ty.FriendlyName()))
	}
}

func flatmapValueFromHCL2Primitive(m map[string]string, key string, val cty.Value) {
	if !val.IsKnown() {
		m[key] = UnknownVariableValue
		return
	}
	if val.IsNull() {
		// Omit entirely
		return
	}

	var err error
	val, err = convert.Convert(val, cty.String)
	if err != nil {
		// Should not be possible, since all primitive types can convert to string.
		panic(fmt.Sprintf("invalid primitive encoding to flatmap: %s", err))
	}
	m[key] = val.AsString()
}

func flatmapValueFromHCL2Map(m map[string]string, prefix string, val cty.Value) {
	if !val.IsKnown() {
		switch {
		case val.Type().IsObjectType():
			// Whole objects can't be unknown in flatmap, so instead we'll
			// just write all of the attribute values out as unknown.
			for name, aty := range val.Type().AttributeTypes() {
				flatmapValueFromHCL2Value(m, prefix+name, cty.UnknownVal(aty))
			}
		default:
			m[prefix+"%"] = UnknownVariableValue
		}
		return
	}

	len := 0
	for it := val.ElementIterator(); it.Next(); {
		ak, av := it.Element()
		name := ak.AsString()
		flatmapValueFromHCL2Value(m, prefix+name, av)
		len++
	}
	if !val.Type().IsObjectType() { // objects don't have an explicit count included, since their attribute count is fixed
		m[prefix+"%"] = strconv.Itoa(len)
	}
}

func flatmapValueFromHCL2Seq(m map[string]string, prefix string, val cty.Value) {
	if !val.IsKnown() {
		m[prefix+"#"] = UnknownVariableValue
		return
	}

	// For sets this won't actually generate exactly what helper/schema would've
	// generated, because we don't have access to the set key function it
	// would've used. However, in practice it doesn't actually matter what the
	// keys are as long as they are unique, so we'll just generate sequential
	// indexes for them as if it were a list.
	//
	// An important implication of this, however, is that the set ordering will
	// not be consistent across mutations and so different keys may be assigned
	// to the same value when round-tripping. Since this shim is intended to
	// be short-lived and not used for round-tripping, we accept this.
	i := 0
	for it := val.ElementIterator(); it.Next(); {
		_, av := it.Element()
		key := prefix + strconv.Itoa(i)
		flatmapValueFromHCL2Value(m, key, av)
		i++
	}
	m[prefix+"#"] = strconv.Itoa(i)
}

// HCL2ValueFromFlatmap converts a map compatible with what would be produced
// by the "flatmap" package to a HCL2 (really, the cty dynamic types library
// that HCL2 uses) object type.
//
// The intended result type must be provided in order to guide how the
// map contents are decoded. This must be an object type or this function
// will panic.
//
// Flatmap values can only represent maps when they are of primitive types,
// so the given type must not have any maps of complex types or the result
// is undefined.
//
// The result may contain null values if the given map does not contain keys
// for all of the different key paths implied by the given type.
func HCL2ValueFromFlatmap(m map[string]string, ty cty.Type) (cty.Value, error) {
	if m == nil {
		return cty.NullVal(ty), nil
	}
	if !ty.IsObjectType() {
		panic(fmt.Sprintf("HCL2ValueFromFlatmap called on %#v", ty))
	}

	return hcl2ValueFromFlatmapObject(m, "", ty.AttributeTypes())
}

func hcl2ValueFromFlatmapValue(m map[string]string, key string, ty cty.Type) (cty.Value, error) {
	var val cty.Value
	var err error
	switch {
	case ty.IsPrimitiveType():
		val, err = hcl2ValueFromFlatmapPrimitive(m, key, ty)
	case ty.IsObjectType():
		val, err = hcl2ValueFromFlatmapObject(m, key+".", ty.AttributeTypes())
	case ty.IsTupleType():
		val, err = hcl2ValueFromFlatmapTuple(m, key+".", ty.TupleElementTypes())
	case ty.IsMapType():
		val, err = hcl2ValueFromFlatmapMap(m, key+".", ty)
	case ty.IsListType():
		val, err = hcl2ValueFromFlatmapList(m, key+".", ty)
	case ty.IsSetType():
		val, err = hcl2ValueFromFlatmapSet(m, key+".", ty)
	default:
		err = fmt.Errorf("cannot decode %s from flatmap", ty.FriendlyName())
	}

	if err != nil {
		return cty.DynamicVal, err
	}
	return val, nil
}

func hcl2ValueFromFlatmapPrimitive(m map[string]string, key string, ty cty.Type) (cty.Value, error) {
	rawVal, exists := m[key]
	if !exists {
		return cty.NullVal(ty), nil
	}
	if rawVal == UnknownVariableValue {
		return cty.UnknownVal(ty), nil
	}

	var err error
	val := cty.StringVal(rawVal)
	val, err = convert.Convert(val, ty)
	if err != nil {
		// This should never happen for _valid_ input, but flatmap data might
		// be tampered with by the user and become invalid.
		return cty.DynamicVal, fmt.Errorf("invalid value for %q in state: %s", key, err)
	}

	return val, nil
}

func hcl2ValueFromFlatmapObject(m map[string]string, prefix string, atys map[string]cty.Type) (cty.Value, error) {
	vals := make(map[string]cty.Value)
	for name, aty := range atys {
		val, err := hcl2ValueFromFlatmapValue(m, prefix+name, aty)
		if err != nil {
			return cty.DynamicVal, err
		}
		vals[name] = val
	}
	return cty.ObjectVal(vals), nil
}

func hcl2ValueFromFlatmapTuple(m map[string]string, prefix string, etys []cty.Type) (cty.Value, error) {
	var vals []cty.Value

	countStr, exists := m[prefix+"#"]
	if !exists {
		return cty.NullVal(cty.Tuple(etys)), nil
	}
	if countStr == UnknownVariableValue {
		return cty.UnknownVal(cty.Tuple(etys)), nil
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return cty.DynamicVal, fmt.Errorf("invalid count value for %q in state: %s", prefix, err)
	}
	if count != len(etys) {
		return cty.DynamicVal, fmt.Errorf("wrong number of values for %q in state: got %d, but need %d", prefix, count, len(etys))
	}

	vals = make([]cty.Value, len(etys))
	for i, ety := range etys {
		key := prefix + strconv.Itoa(i)
		val, err := hcl2ValueFromFlatmapValue(m, key, ety)
		if err != nil {
			return cty.DynamicVal, err
		}
		vals[i] = val
	}
	return cty.TupleVal(vals), nil
}

func hcl2ValueFromFlatmapMap(m map[string]string, prefix string, ty cty.Type) (cty.Value, error) {
	vals := make(map[string]cty.Value)
	ety := ty.ElementType()

	// We actually don't really care about the "count" of a map for our
	// purposes here, but we do need to check if it _exists_ in order to
	// recognize the difference between null (not set at all) and empty.
	if strCount, exists := m[prefix+"%"]; !exists {
		return cty.NullVal(ty), nil
	} else if strCount == UnknownVariableValue {
		return cty.UnknownVal(ty), nil
	}

	for fullKey := range m {
		if !strings.HasPrefix(fullKey, prefix) {
			continue
		}

		// The flatmap format doesn't allow us to distinguish between keys
		// that contain periods and nested objects, so by convention a
		// map is only ever of primitive type in flatmap, and we just assume
		// that the remainder of the raw key (dots and all) is the key we
		// want in the result value.
		key := fullKey[len(prefix):]
		if key == "%" {
			// Ignore the "count" key
			continue
		}

		val, err := hcl2ValueFromFlatmapValue(m, fullKey, ety)
		if err != nil {
			return cty.DynamicVal, err
		}
		vals[key] = val
	}

	if len(vals) == 0 {
		return cty.MapValEmpty(ety), nil
	}
	return cty.MapVal(vals), nil
}

func hcl2ValueFromFlatmapList(m map[string]string, prefix string, ty cty.Type) (cty.Value, error) {
	var vals []cty.Value

	countStr, exists := m[prefix+"#"]
	if !exists {
		return cty.NullVal(ty), nil
	}
	if countStr == UnknownVariableValue {
		return cty.UnknownVal(ty), nil
	}

	count, err := strconv.Atoi(countStr)
	if err != nil {
		return cty.DynamicVal, fmt.Errorf("invalid count value for %q in state: %s", prefix, err)
	}

	ety := ty.ElementType()
	if count == 0 {
		return cty.ListValEmpty(ety), nil
	}

	vals = make([]cty.Value, count)
	for i := 0; i < count; i++ {
		key := prefix + strconv.Itoa(i)
		val, err := hcl2ValueFromFlatmapValue(m, key, ety)
		if err != nil {
			return cty.DynamicVal, err
		}
		vals[i] = val
	}

	return cty.ListVal(vals), nil
}

func hcl2ValueFromFlatmapSet(m map[string]string, prefix string, ty cty.Type) (cty.Value, error) {
	var vals []cty.Value
	ety := ty.ElementType()

	// We actually don't really care about the "count" of a set for our
	// purposes here, but we do need to check if it _exists_ in order to
	// recognize the difference between null (not set at all) and empty.
	if strCount, exists := m[prefix+"#"]; !exists {
		return cty.NullVal(ty), nil
	} else if strCount == UnknownVariableValue {
		return cty.UnknownVal(ty), nil
	}

	for fullKey := range m {
		if !strings.HasPrefix(fullKey, prefix) {
			continue
		}
		subKey := fullKey[len(prefix):]
		if subKey == "#" {
			// Ignore the "count" key
			continue
		}
		key := fullKey
		if dot := strings.IndexByte(subKey, '.'); dot != -1 {
			key = fullKey[:dot+len(prefix)]
		}

		// The flatmap format doesn't allow us to distinguish between keys
		// that contain periods and nested objects, so by convention a
		// map is only ever of primitive type in flatmap, and we just assume
		// that the remainder of the raw key (dots and all) is the key we
		// want in the result value.

		val, err := hcl2ValueFromFlatmapValue(m, key, ety)
		if err != nil {
			return cty.DynamicVal, err
		}
		vals = append(vals, val)
	}

	if len(vals) == 0 {
		return cty.SetValEmpty(ety), nil
	}
	return cty.SetVal(vals), nil
}
