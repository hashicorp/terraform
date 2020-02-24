package msgpack

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/vmihailenco/msgpack"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// Marshal produces a msgpack serialization of the given value that
// can be decoded into the given type later using Unmarshal.
//
// The given value must conform to the given type, or an error will
// be returned.
func Marshal(val cty.Value, ty cty.Type) ([]byte, error) {
	errs := val.Type().TestConformance(ty)
	if errs != nil {
		// Attempt a conversion
		var err error
		val, err = convert.Convert(val, ty)
		if err != nil {
			return nil, err
		}
	}

	// From this point onward, val can be assumed to be conforming to t.

	var path cty.Path
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)

	err := marshal(val, ty, path, enc)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func marshal(val cty.Value, ty cty.Type, path cty.Path, enc *msgpack.Encoder) error {
	if val.IsMarked() {
		return path.NewErrorf("value has marks, so it cannot be seralized")
	}

	// If we're going to decode as DynamicPseudoType then we need to save
	// dynamic type information to recover the real type.
	if ty == cty.DynamicPseudoType && val.Type() != cty.DynamicPseudoType {
		return marshalDynamic(val, path, enc)
	}

	if !val.IsKnown() {
		err := enc.Encode(unknownVal)
		if err != nil {
			return path.NewError(err)
		}
		return nil
	}
	if val.IsNull() {
		err := enc.EncodeNil()
		if err != nil {
			return path.NewError(err)
		}
		return nil
	}

	// The caller should've guaranteed that the given val is conformant with
	// the given type ty, so we'll proceed under that assumption here.
	switch {
	case ty.IsPrimitiveType():
		switch ty {
		case cty.String:
			err := enc.EncodeString(val.AsString())
			if err != nil {
				return path.NewError(err)
			}
			return nil
		case cty.Number:
			var err error
			switch {
			case val.RawEquals(cty.PositiveInfinity):
				err = enc.EncodeFloat64(positiveInfinity)
			case val.RawEquals(cty.NegativeInfinity):
				err = enc.EncodeFloat64(negativeInfinity)
			default:
				bf := val.AsBigFloat()
				if iv, acc := bf.Int64(); acc == big.Exact {
					err = enc.EncodeInt(iv)
				} else if fv, acc := bf.Float64(); acc == big.Exact {
					err = enc.EncodeFloat64(fv)
				} else {
					err = enc.EncodeString(bf.Text('f', -1))
				}
			}
			if err != nil {
				return path.NewError(err)
			}
			return nil
		case cty.Bool:
			err := enc.EncodeBool(val.True())
			if err != nil {
				return path.NewError(err)
			}
			return nil
		default:
			panic("unsupported primitive type")
		}
	case ty.IsListType(), ty.IsSetType():
		enc.EncodeArrayLen(val.LengthInt())
		ety := ty.ElementType()
		it := val.ElementIterator()
		path := append(path, nil) // local override of 'path' with extra element
		for it.Next() {
			ek, ev := it.Element()
			path[len(path)-1] = cty.IndexStep{
				Key: ek,
			}
			err := marshal(ev, ety, path, enc)
			if err != nil {
				return err
			}
		}
		return nil
	case ty.IsMapType():
		enc.EncodeMapLen(val.LengthInt())
		ety := ty.ElementType()
		it := val.ElementIterator()
		path := append(path, nil) // local override of 'path' with extra element
		for it.Next() {
			ek, ev := it.Element()
			path[len(path)-1] = cty.IndexStep{
				Key: ek,
			}
			var err error
			err = marshal(ek, ek.Type(), path, enc)
			if err != nil {
				return err
			}
			err = marshal(ev, ety, path, enc)
			if err != nil {
				return err
			}
		}
		return nil
	case ty.IsTupleType():
		etys := ty.TupleElementTypes()
		it := val.ElementIterator()
		path := append(path, nil) // local override of 'path' with extra element
		i := 0
		enc.EncodeArrayLen(len(etys))
		for it.Next() {
			ety := etys[i]
			ek, ev := it.Element()
			path[len(path)-1] = cty.IndexStep{
				Key: ek,
			}
			err := marshal(ev, ety, path, enc)
			if err != nil {
				return err
			}
			i++
		}
		return nil
	case ty.IsObjectType():
		atys := ty.AttributeTypes()
		path := append(path, nil) // local override of 'path' with extra element

		names := make([]string, 0, len(atys))
		for k := range atys {
			names = append(names, k)
		}
		sort.Strings(names)

		enc.EncodeMapLen(len(names))

		for _, k := range names {
			aty := atys[k]
			av := val.GetAttr(k)
			path[len(path)-1] = cty.GetAttrStep{
				Name: k,
			}
			var err error
			err = marshal(cty.StringVal(k), cty.String, path, enc)
			if err != nil {
				return err
			}
			err = marshal(av, aty, path, enc)
			if err != nil {
				return err
			}
		}
		return nil
	case ty.IsCapsuleType():
		return path.NewErrorf("capsule types not supported for msgpack encoding")
	default:
		// should never happen
		return path.NewErrorf("cannot msgpack-serialize %s", ty.FriendlyName())
	}
}

// marshalDynamic adds an extra wrapping object containing dynamic type
// information for the given value.
func marshalDynamic(val cty.Value, path cty.Path, enc *msgpack.Encoder) error {
	dv := dynamicVal{
		Value: val,
		Path:  path,
	}
	return enc.Encode(&dv)
}
