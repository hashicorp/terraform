package msgpack

import (
	"bytes"
	"math/big"

	"github.com/vmihailenco/msgpack"
	msgpackCodes "github.com/vmihailenco/msgpack/codes"
	"github.com/zclconf/go-cty/cty"
)

// Unmarshal interprets the given bytes as a msgpack-encoded cty Value of
// the given type, returning the result.
//
// If an error is returned, the error is written with a hypothetical
// end-user that wrote the msgpack file as its audience, using cty type
// system concepts rather than Go type system concepts.
func Unmarshal(b []byte, ty cty.Type) (cty.Value, error) {
	r := bytes.NewReader(b)
	dec := msgpack.NewDecoder(r)

	var path cty.Path
	return unmarshal(dec, ty, path)
}

func unmarshal(dec *msgpack.Decoder, ty cty.Type, path cty.Path) (cty.Value, error) {
	peek, err := dec.PeekCode()
	if err != nil {
		return cty.DynamicVal, path.NewError(err)
	}
	if msgpackCodes.IsExt(peek) {
		// We just assume _all_ extensions are unknown values,
		// since we don't have any other extensions.
		dec.Skip() // skip what we've peeked
		return cty.UnknownVal(ty), nil
	}
	if ty == cty.DynamicPseudoType {
		return unmarshalDynamic(dec, path)
	}
	if peek == msgpackCodes.Nil {
		dec.Skip() // skip what we've peeked
		return cty.NullVal(ty), nil
	}

	switch {
	case ty.IsPrimitiveType():
		val, err := unmarshalPrimitive(dec, ty, path)
		if err != nil {
			return cty.NilVal, err
		}
		return val, nil
	case ty.IsListType():
		return unmarshalList(dec, ty.ElementType(), path)
	case ty.IsSetType():
		return unmarshalSet(dec, ty.ElementType(), path)
	case ty.IsMapType():
		return unmarshalMap(dec, ty.ElementType(), path)
	case ty.IsTupleType():
		return unmarshalTuple(dec, ty.TupleElementTypes(), path)
	case ty.IsObjectType():
		return unmarshalObject(dec, ty.AttributeTypes(), path)
	default:
		return cty.NilVal, path.NewErrorf("unsupported type %s", ty.FriendlyName())
	}
}

func unmarshalPrimitive(dec *msgpack.Decoder, ty cty.Type, path cty.Path) (cty.Value, error) {
	switch ty {
	case cty.Bool:
		rv, err := dec.DecodeBool()
		if err != nil {
			return cty.DynamicVal, path.NewErrorf("bool is required")
		}
		return cty.BoolVal(rv), nil
	case cty.Number:
		// Marshal will try int and float first, if the value can be
		// losslessly represented in these encodings, and then fall
		// back on a string if the number is too large or too precise.
		peek, err := dec.PeekCode()
		if err != nil {
			return cty.DynamicVal, path.NewErrorf("number is required")
		}

		if msgpackCodes.IsFixedNum(peek) {
			rv, err := dec.DecodeInt64()
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			return cty.NumberIntVal(rv), nil
		}

		switch peek {
		case msgpackCodes.Int8, msgpackCodes.Int16, msgpackCodes.Int32, msgpackCodes.Int64:
			rv, err := dec.DecodeInt64()
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			return cty.NumberIntVal(rv), nil
		case msgpackCodes.Uint8, msgpackCodes.Uint16, msgpackCodes.Uint32, msgpackCodes.Uint64:
			rv, err := dec.DecodeUint64()
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			return cty.NumberUIntVal(rv), nil
		case msgpackCodes.Float, msgpackCodes.Double:
			rv, err := dec.DecodeFloat64()
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			return cty.NumberFloatVal(rv), nil
		default:
			rv, err := dec.DecodeString()
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			bf := &big.Float{}
			_, _, err = bf.Parse(rv, 10)
			if err != nil {
				return cty.DynamicVal, path.NewErrorf("number is required")
			}
			return cty.NumberVal(bf), nil
		}
	case cty.String:
		rv, err := dec.DecodeString()
		if err != nil {
			return cty.DynamicVal, path.NewErrorf("string is required")
		}
		return cty.StringVal(rv), nil
	default:
		// should never happen
		panic("unsupported primitive type")
	}
}

func unmarshalList(dec *msgpack.Decoder, ety cty.Type, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeArrayLen()
	if err != nil {
		return cty.DynamicVal, path.NewErrorf("a list is required")
	}

	switch {
	case length < 0:
		return cty.NullVal(cty.List(ety)), nil
	case length == 0:
		return cty.ListValEmpty(ety), nil
	}

	vals := make([]cty.Value, 0, length)
	path = append(path, nil)
	for i := 0; i < length; i++ {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.NumberIntVal(int64(i)),
		}

		val, err := unmarshal(dec, ety, path)
		if err != nil {
			return cty.DynamicVal, err
		}

		vals = append(vals, val)
	}

	return cty.ListVal(vals), nil
}

func unmarshalSet(dec *msgpack.Decoder, ety cty.Type, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeArrayLen()
	if err != nil {
		return cty.DynamicVal, path.NewErrorf("a set is required")
	}

	switch {
	case length < 0:
		return cty.NullVal(cty.Set(ety)), nil
	case length == 0:
		return cty.SetValEmpty(ety), nil
	}

	vals := make([]cty.Value, 0, length)
	path = append(path, nil)
	for i := 0; i < length; i++ {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.NumberIntVal(int64(i)),
		}

		val, err := unmarshal(dec, ety, path)
		if err != nil {
			return cty.DynamicVal, err
		}

		vals = append(vals, val)
	}

	return cty.SetVal(vals), nil
}

func unmarshalMap(dec *msgpack.Decoder, ety cty.Type, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeMapLen()
	if err != nil {
		return cty.DynamicVal, path.NewErrorf("a map is required")
	}

	switch {
	case length < 0:
		return cty.NullVal(cty.Map(ety)), nil
	case length == 0:
		return cty.MapValEmpty(ety), nil
	}

	vals := make(map[string]cty.Value, length)
	path = append(path, nil)
	for i := 0; i < length; i++ {
		key, err := dec.DecodeString()
		if err != nil {
			path[:len(path)-1].NewErrorf("non-string key in map")
		}

		path[len(path)-1] = cty.IndexStep{
			Key: cty.StringVal(key),
		}

		val, err := unmarshal(dec, ety, path)
		if err != nil {
			return cty.DynamicVal, err
		}

		vals[key] = val
	}

	return cty.MapVal(vals), nil
}

func unmarshalTuple(dec *msgpack.Decoder, etys []cty.Type, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeArrayLen()
	if err != nil {
		return cty.DynamicVal, path.NewErrorf("a tuple is required")
	}

	switch {
	case length < 0:
		return cty.NullVal(cty.Tuple(etys)), nil
	case length == 0:
		return cty.TupleVal(nil), nil
	case length != len(etys):
		return cty.DynamicVal, path.NewErrorf("a tuple of length %d is required", len(etys))
	}

	vals := make([]cty.Value, 0, length)
	path = append(path, nil)
	for i := 0; i < length; i++ {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.NumberIntVal(int64(i)),
		}
		ety := etys[i]

		val, err := unmarshal(dec, ety, path)
		if err != nil {
			return cty.DynamicVal, err
		}

		vals = append(vals, val)
	}

	return cty.TupleVal(vals), nil
}

func unmarshalObject(dec *msgpack.Decoder, atys map[string]cty.Type, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeMapLen()
	if err != nil {
		return cty.DynamicVal, path.NewErrorf("an object is required")
	}

	switch {
	case length < 0:
		return cty.NullVal(cty.Object(atys)), nil
	case length == 0:
		return cty.ObjectVal(nil), nil
	case length != len(atys):
		return cty.DynamicVal, path.NewErrorf("an object with %d attributes is required (%d given)",
			len(atys), length)
	}

	vals := make(map[string]cty.Value, length)
	path = append(path, nil)
	for i := 0; i < length; i++ {
		key, err := dec.DecodeString()
		if err != nil {
			return cty.DynamicVal, path[:len(path)-1].NewErrorf("all keys must be strings")
		}

		path[len(path)-1] = cty.IndexStep{
			Key: cty.StringVal(key),
		}
		aty, exists := atys[key]
		if !exists {
			return cty.DynamicVal, path.NewErrorf("unsupported attribute")
		}

		val, err := unmarshal(dec, aty, path)
		if err != nil {
			return cty.DynamicVal, err
		}

		vals[key] = val
	}

	return cty.ObjectVal(vals), nil
}

func unmarshalDynamic(dec *msgpack.Decoder, path cty.Path) (cty.Value, error) {
	length, err := dec.DecodeArrayLen()
	if err != nil {
		return cty.DynamicVal, path.NewError(err)
	}

	switch {
	case length == -1:
		return cty.NullVal(cty.DynamicPseudoType), nil
	case length != 2:
		return cty.DynamicVal, path.NewErrorf(
			"dynamic value array must have exactly two elements",
		)
	}

	typeJSON, err := dec.DecodeBytes()
	if err != nil {
		return cty.DynamicVal, path.NewError(err)
	}
	var ty cty.Type
	err = (&ty).UnmarshalJSON(typeJSON)
	if err != nil {
		return cty.DynamicVal, path.NewError(err)
	}

	return unmarshal(dec, ty, path)
}
