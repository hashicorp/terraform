package cty

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"math/big"

	"github.com/zclconf/go-cty/cty/set"
)

// GobEncode is an implementation of the gob.GobEncoder interface, which
// allows Values to be included in structures encoded with encoding/gob.
//
// Currently it is not possible to represent values of capsule types in gob,
// because the types themselves cannot be represented.
func (val Value) GobEncode() ([]byte, error) {
	if val.IsMarked() {
		return nil, errors.New("value is marked")
	}

	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)

	gv := gobValue{
		Version: 0,
		Ty:      val.ty,
		V:       val.v,
	}

	err := enc.Encode(gv)
	if err != nil {
		return nil, fmt.Errorf("error encoding cty.Value: %s", err)
	}

	return buf.Bytes(), nil
}

// GobDecode is an implementation of the gob.GobDecoder interface, which
// inverts the operation performed by GobEncode. See the documentation of
// GobEncode for considerations when using cty.Value instances with gob.
func (val *Value) GobDecode(buf []byte) error {
	r := bytes.NewReader(buf)
	dec := gob.NewDecoder(r)

	var gv gobValue
	err := dec.Decode(&gv)
	if err != nil {
		return fmt.Errorf("error decoding cty.Value: %s", err)
	}
	if gv.Version != 0 {
		return fmt.Errorf("unsupported cty.Value encoding version %d; only 0 is supported", gv.Version)
	}

	// Because big.Float.GobEncode is implemented with a pointer reciever,
	// gob encoding of an interface{} containing a *big.Float value does not
	// round-trip correctly, emerging instead as a non-pointer big.Float.
	// The rest of cty expects all number values to be represented by
	// *big.Float, so we'll fix that up here.
	gv.V = gobDecodeFixNumberPtr(gv.V, gv.Ty)

	val.ty = gv.Ty
	val.v = gv.V

	return nil
}

// GobEncode is an implementation of the gob.GobEncoder interface, which
// allows Types to be included in structures encoded with encoding/gob.
//
// Currently it is not possible to represent capsule types in gob.
func (t Type) GobEncode() ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)

	gt := gobType{
		Version: 0,
		Impl:    t.typeImpl,
	}

	err := enc.Encode(gt)
	if err != nil {
		return nil, fmt.Errorf("error encoding cty.Type: %s", err)
	}

	return buf.Bytes(), nil
}

// GobDecode is an implementatino of the gob.GobDecoder interface, which
// reverses the encoding performed by GobEncode to allow types to be recovered
// from gob buffers.
func (t *Type) GobDecode(buf []byte) error {
	r := bytes.NewReader(buf)
	dec := gob.NewDecoder(r)

	var gt gobType
	err := dec.Decode(&gt)
	if err != nil {
		return fmt.Errorf("error decoding cty.Type: %s", err)
	}
	if gt.Version != 0 {
		return fmt.Errorf("unsupported cty.Type encoding version %d; only 0 is supported", gt.Version)
	}

	t.typeImpl = gt.Impl

	return nil
}

// Capsule types cannot currently be gob-encoded, because they rely on pointer
// equality and we have no way to recover the original pointer on decode.
func (t *capsuleType) GobEncode() ([]byte, error) {
	return nil, fmt.Errorf("cannot gob-encode capsule type %q", t.FriendlyName(friendlyTypeName))
}

func (t *capsuleType) GobDecode() ([]byte, error) {
	return nil, fmt.Errorf("cannot gob-decode capsule type %q", t.FriendlyName(friendlyTypeName))
}

type gobValue struct {
	Version int
	Ty      Type
	V       interface{}
}

type gobType struct {
	Version int
	Impl    typeImpl
}

type gobCapsuleTypeImpl struct {
}

// goDecodeFixNumberPtr fixes an unfortunate quirk of round-tripping cty.Number
// values through gob: the big.Float.GobEncode method is implemented on a
// pointer receiver, and so it loses the "pointer-ness" of the value on
// encode, causing the values to emerge the other end as big.Float rather than
// *big.Float as we expect elsewhere in cty.
//
// The implementation of gobDecodeFixNumberPtr mutates the given raw value
// during its work, and may either return the same value mutated or a new
// value. Callers must no longer use whatever value they pass as "raw" after
// this function is called.
func gobDecodeFixNumberPtr(raw interface{}, ty Type) interface{} {
	// Unfortunately we need to work recursively here because number values
	// might be embedded in structural or collection type values.

	switch {
	case ty.Equals(Number):
		if bf, ok := raw.(big.Float); ok {
			return &bf // wrap in pointer
		}
	case ty.IsMapType() && ty.ElementType().Equals(Number):
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				m[k] = gobDecodeFixNumberPtr(v, ty.ElementType())
			}
		}
	case ty.IsListType() && ty.ElementType().Equals(Number):
		if s, ok := raw.([]interface{}); ok {
			for i, v := range s {
				s[i] = gobDecodeFixNumberPtr(v, ty.ElementType())
			}
		}
	case ty.IsSetType() && ty.ElementType().Equals(Number):
		if s, ok := raw.(set.Set); ok {
			newS := set.NewSet(s.Rules())
			for it := s.Iterator(); it.Next(); {
				newV := gobDecodeFixNumberPtr(it.Value(), ty.ElementType())
				newS.Add(newV)
			}
			return newS
		}
	case ty.IsObjectType():
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				aty := ty.AttributeType(k)
				m[k] = gobDecodeFixNumberPtr(v, aty)
			}
		}
	case ty.IsTupleType():
		if s, ok := raw.([]interface{}); ok {
			for i, v := range s {
				ety := ty.TupleElementType(i)
				s[i] = gobDecodeFixNumberPtr(v, ety)
			}
		}
	}

	return raw
}

// gobDecodeFixNumberPtrVal is a helper wrapper around gobDecodeFixNumberPtr
// that works with already-constructed values. This is primarily for testing,
// to fix up intentionally-invalid number values for the parts of the test
// code that need them to be valid, such as calling GoString on them.
func gobDecodeFixNumberPtrVal(v Value) Value {
	raw := gobDecodeFixNumberPtr(v.v, v.ty)
	return Value{
		v:  raw,
		ty: v.ty,
	}
}
