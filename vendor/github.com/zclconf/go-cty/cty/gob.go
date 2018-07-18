package cty

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"
)

// GobEncode is an implementation of the gob.GobEncoder interface, which
// allows Values to be included in structures encoded with encoding/gob.
//
// Currently it is not possible to represent values of capsule types in gob,
// because the types themselves cannot be represented.
func (val Value) GobEncode() ([]byte, error) {
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

	// big.Float seems to, for some reason, lose its "pointerness" when we
	// round-trip it, so we'll fix that here.
	if bf, ok := gv.V.(big.Float); ok {
		gv.V = &bf
	}

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
