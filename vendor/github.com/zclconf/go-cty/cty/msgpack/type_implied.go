package msgpack

import (
	"bytes"
	"fmt"
	"io"

	"github.com/vmihailenco/msgpack/v4"
	msgpackcodes "github.com/vmihailenco/msgpack/v4/codes"
	"github.com/zclconf/go-cty/cty"
)

// ImpliedType returns the cty Type implied by the structure of the given
// msgpack-compliant buffer. This function implements the default type mapping
// behavior used when decoding arbitrary msgpack without explicit cty Type
// information.
//
// The rules are as follows:
//
// msgpack strings, numbers and bools map to their equivalent primitive type in
// cty.
//
// msgpack maps become cty object types, with the attributes defined by the
// map keys and the types of their values.
//
// msgpack arrays become cty tuple types, with the elements defined by the
// types of the array members.
//
// Any nulls are typed as DynamicPseudoType, so callers of this function
// must be prepared to deal with this. Callers that do not wish to deal with
// dynamic typing should not use this function and should instead describe
// their required types explicitly with a cty.Type instance when decoding.
//
// Any unknown values are similarly typed as DynamicPseudoType, because these
// do not carry type information on the wire.
//
// Any parse errors will be returned as an error, and the type will be the
// invalid value cty.NilType.
func ImpliedType(buf []byte) (cty.Type, error) {
	r := bytes.NewReader(buf)
	dec := msgpack.NewDecoder(r)

	ty, err := impliedType(dec)
	if err != nil {
		return cty.NilType, err
	}

	// We must now be at the end of the buffer
	err = dec.Skip()
	if err != io.EOF {
		return ty, fmt.Errorf("extra bytes after msgpack value")
	}

	return ty, nil
}

func impliedType(dec *msgpack.Decoder) (cty.Type, error) {
	// If this function returns with a nil error then it must have already
	// consumed the next value from the decoder, since when called recursively
	// the caller will be expecting to find a following value here.

	code, err := dec.PeekCode()
	if err != nil {
		return cty.NilType, err
	}

	switch {

	case code == msgpackcodes.Nil || msgpackcodes.IsExt(code):
		err := dec.Skip()
		return cty.DynamicPseudoType, err

	case code == msgpackcodes.True || code == msgpackcodes.False:
		_, err := dec.DecodeBool()
		return cty.Bool, err

	case msgpackcodes.IsFixedNum(code):
		_, err := dec.DecodeInt64()
		return cty.Number, err

	case code == msgpackcodes.Int8 || code == msgpackcodes.Int16 || code == msgpackcodes.Int32 || code == msgpackcodes.Int64:
		_, err := dec.DecodeInt64()
		return cty.Number, err

	case code == msgpackcodes.Uint8 || code == msgpackcodes.Uint16 || code == msgpackcodes.Uint32 || code == msgpackcodes.Uint64:
		_, err := dec.DecodeUint64()
		return cty.Number, err

	case code == msgpackcodes.Float || code == msgpackcodes.Double:
		_, err := dec.DecodeFloat64()
		return cty.Number, err

	case msgpackcodes.IsString(code):
		_, err := dec.DecodeString()
		return cty.String, err

	case msgpackcodes.IsFixedMap(code) || code == msgpackcodes.Map16 || code == msgpackcodes.Map32:
		return impliedObjectType(dec)

	case msgpackcodes.IsFixedArray(code) || code == msgpackcodes.Array16 || code == msgpackcodes.Array32:
		return impliedTupleType(dec)

	default:
		return cty.NilType, fmt.Errorf("unsupported msgpack code %#v", code)
	}
}

func impliedObjectType(dec *msgpack.Decoder) (cty.Type, error) {
	// If we get in here then we've already peeked the next code and know
	// it's some sort of map.
	l, err := dec.DecodeMapLen()
	if err != nil {
		return cty.DynamicPseudoType, nil
	}

	var atys map[string]cty.Type

	for i := 0; i < l; i++ {
		// Read the map key first. We require maps to be strings, but msgpack
		// doesn't so we're prepared to error here if not.
		k, err := dec.DecodeString()
		if err != nil {
			return cty.DynamicPseudoType, err
		}

		aty, err := impliedType(dec)
		if err != nil {
			return cty.DynamicPseudoType, err
		}

		if atys == nil {
			atys = make(map[string]cty.Type)
		}
		atys[k] = aty
	}

	if len(atys) == 0 {
		return cty.EmptyObject, nil
	}

	return cty.Object(atys), nil
}

func impliedTupleType(dec *msgpack.Decoder) (cty.Type, error) {
	// If we get in here then we've already peeked the next code and know
	// it's some sort of array.
	l, err := dec.DecodeArrayLen()
	if err != nil {
		return cty.DynamicPseudoType, nil
	}

	if l == 0 {
		return cty.EmptyTuple, nil
	}

	etys := make([]cty.Type, l)

	for i := 0; i < l; i++ {
		ety, err := impliedType(dec)
		if err != nil {
			return cty.DynamicPseudoType, err
		}
		etys[i] = ety
	}

	return cty.Tuple(etys), nil
}
