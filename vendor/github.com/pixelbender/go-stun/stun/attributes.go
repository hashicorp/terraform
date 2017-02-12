package stun

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
)

// Attributes introduced by the RFC 5389 Section 18.2.
const (
	AttrMappedAddress     = uint16(0x0001)
	AttrXorMappedAddress  = uint16(0x0020)
	AttrUsername          = uint16(0x0006)
	AttrMessageIntegrity  = uint16(0x0008)
	AttrErrorCode         = uint16(0x0009)
	AttrRealm             = uint16(0x0014)
	AttrNonce             = uint16(0x0015)
	AttrUnknownAttributes = uint16(0x000a)
	AttrSoftware          = uint16(0x8022)
	AttrAlternateServer   = uint16(0x8023)
	AttrFingerprint       = uint16(0x8028)
)

// Attributes introduced by the RFC 5780 Section 7.
const (
	AttrChangeRequest  = uint16(0x0003)
	AttrPadding        = uint16(0x0026)
	AttrResponsePort   = uint16(0x0027)
	AttrResponseOrigin = uint16(0x802b)
	AttrOtherAddress   = uint16(0x802c)
)

// Attributes introduced by the RFC 3489 Section 11.2 except listed above.
const (
	AttrResponseAddress = uint16(0x0002)
	AttrSourceAddress   = uint16(0x0004)
	AttrChangedAddress  = uint16(0x0005)
	AttrPassword        = uint16(0x0007)
	AttrReflectedFrom   = uint16(0x000b)
)

// Bits definition of CHANGE-REQUEST attribute.
const (
	ChangeIP   = uint32(0x4)
	ChangePort = uint32(0x2)
)

var attrNames = map[uint16]string{
	AttrMappedAddress:     "MAPPED-ADDRESS",
	AttrXorMappedAddress:  "XOR-MAPPED-ADDRESS",
	AttrUsername:          "USERNAME",
	AttrMessageIntegrity:  "MESSAGE-INTEGRITY",
	AttrFingerprint:       "FINGERPRINT",
	AttrErrorCode:         "ERROR-CODE",
	AttrRealm:             "REALM",
	AttrNonce:             "NONCE",
	AttrUnknownAttributes: "UNKNOWN-ATTRIBUTES",
	AttrSoftware:          "SOFTWARE",
	AttrAlternateServer:   "ALTERNATE-SERVER",
	AttrChangeRequest:     "CHANGE-REQUEST",
	AttrPadding:           "PADDING",
	AttrResponsePort:      "RESPONSE-PORT",
	AttrResponseOrigin:    "RESPONSE-ORIGIN",
	AttrOtherAddress:      "OTHER-ADDRESS",
	AttrResponseAddress:   "RESPONSE-ADDRESS",
	AttrSourceAddress:     "SOURCE-ADDRESS",
	AttrChangedAddress:    "CHANGED-ADDRESS",
	AttrPassword:          "PASSWORD",
	AttrReflectedFrom:     "REFLECTED-FROM",
}

// GetAttributeName returns a STUN attribute name.
// It returns the empty string if the attribute is unknown.
func GetAttributeName(at uint16) (n string) {
	if n = attrNames[at]; n == "" {
		n = "0x" + strconv.FormatUint(uint64(at), 16)
	}
	return
}

// AttrCodec interface represents a STUN attribute encoder/decoder.
type AttrCodec interface {
	Encode(w Writer, v interface{}) error
	Decode(r Reader) (interface{}, error)
}

var attrCodecs = map[uint16]AttrCodec{
	AttrMappedAddress:     AddrCodec,
	AttrXorMappedAddress:  XorAddrCodec,
	AttrUsername:          StringCodec,
	AttrMessageIntegrity:  RawCodec,
	AttrErrorCode:         errorCodec{},
	AttrRealm:             StringCodec,
	AttrNonce:             StringCodec,
	AttrUnknownAttributes: unkAttrCodec{},
	AttrSoftware:          StringCodec,
	AttrAlternateServer:   AddrCodec,
	AttrFingerprint:       uint32Codec{},
	AttrChangeRequest:     uint32Codec{},
	AttrPadding:           RawCodec,
	AttrResponsePort:      portCodec{},
	AttrResponseOrigin:    AddrCodec,
	AttrOtherAddress:      AddrCodec,
	AttrResponseAddress:   AddrCodec,
	AttrSourceAddress:     AddrCodec,
	AttrChangedAddress:    AddrCodec,
	AttrPassword:          StringCodec,
	AttrReflectedFrom:     AddrCodec,
}

// GetAttributeCodec returns a STUN attribute codec for TURN.
// It returns the nil if the attribute type is unknown.
func GetAttributeCodec(at uint16) AttrCodec {
	return attrCodecs[at]
}

type errUnsupportedAttrType struct {
	reflect.Type
}

func (err errUnsupportedAttrType) Error() string {
	return "stun: unsupported attribute type: " + reflect.Type(err).String()
}

// ErrUnknownAttrs is returned when a STUN message contains unknown comprehension-required attributes.
type errUnknownAttrCodec struct {
	Type uint16
}

func (err errUnknownAttrCodec) Error() string {
	return "stun: no codec for attribute " + GetAttributeName(err.Type) + " is defined"
}

// Attributes represents a set of STUN attributes.
type Attributes map[uint16]interface{}

func (at Attributes) Has(id uint16) (ok bool) {
	_, ok = at[id]
	return
}

func (at Attributes) String(id uint16) string {
	r, ok := at[id]
	if ok {
		switch v := r.(type) {
		case []byte:
			return string(v)
		case string:
			return v
		case (fmt.Stringer):
			return v.String()
		default:
			return fmt.Sprintf("%", r)
		}
	}
	return ""
}

// RawCodec encodes and decodes a STUN attribute as a raw byte array.
const RawCodec = rawCodec(false)

// StringCodec encodes and decodes a STUN attribute as a string.
const StringCodec = rawCodec(true)

type rawCodec bool

func (c rawCodec) Encode(w Writer, v interface{}) error {
	switch attr := v.(type) {
	case []byte:
		copy(w.Next(len(attr)), attr)
	case string:
		copy(w.Next(len(attr)), attr)
	default:
		return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
	}
	return nil
}

func (c rawCodec) Decode(r Reader) (interface{}, error) {
	b, _ := r.Next(r.Available())
	if c {
		return string(b), nil
	}
	return b, nil
}

type unkAttrCodec struct{}

func (c unkAttrCodec) Encode(w Writer, v interface{}) error {
	switch attr := v.(type) {
	case []uint16:
		c.writeAttributeTypes(w, attr)
	case ErrUnknownAttrs:
		c.writeAttributeTypes(w, attr.Attributes)
	default:
		return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
	}
	return nil
}

func (c unkAttrCodec) writeAttributeTypes(w Writer, attrs []uint16) {
	b := w.Next(len(attrs) << 1)
	for i, it := range attrs {
		be.PutUint16(b[i<<1:], it)
	}
}

func (c unkAttrCodec) Decode(r Reader) (interface{}, error) {
	b, _ := r.Next(r.Available())
	attrs := make([]uint16, len(b)>>1)
	for i := range attrs {
		attrs[i] = be.Uint16(b[i<<1:])
	}
	return &ErrUnknownAttrs{attrs}, nil
}

type uint32Codec struct{}

func (c uint32Codec) Encode(w Writer, v interface{}) error {
	if v, ok := v.(uint32); ok {
		be.PutUint32(w.Next(4), v)
		return nil
	}
	return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
}

func (c uint32Codec) Decode(r Reader) (interface{}, error) {
	b, err := r.Next(4)
	if err != nil {
		return nil, err
	}
	return be.Uint32(b), nil
}

type portCodec struct{}

func (c portCodec) Encode(w Writer, v interface{}) error {
	if v, ok := v.(int); ok {
		be.PutUint16(w.Next(2), uint16(v))
		return nil
	}
	return &errUnsupportedAttrType{Type: reflect.TypeOf(v)}
}

func (c portCodec) Decode(r Reader) (interface{}, error) {
	b, err := r.Next(2)
	if err != nil {
		return nil, err
	}
	return int(be.Uint16(b)), nil
}

var be = binary.BigEndian
