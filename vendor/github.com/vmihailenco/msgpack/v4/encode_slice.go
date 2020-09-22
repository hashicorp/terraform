package msgpack

import (
	"reflect"

	"github.com/vmihailenco/msgpack/v4/codes"
)

var sliceStringType = reflect.TypeOf(([]string)(nil))

func encodeStringValue(e *Encoder, v reflect.Value) error {
	return e.EncodeString(v.String())
}

func encodeByteSliceValue(e *Encoder, v reflect.Value) error {
	return e.EncodeBytes(v.Bytes())
}

func encodeByteArrayValue(e *Encoder, v reflect.Value) error {
	if err := e.EncodeBytesLen(v.Len()); err != nil {
		return err
	}

	if v.CanAddr() {
		b := v.Slice(0, v.Len()).Bytes()
		return e.write(b)
	}

	e.buf = grow(e.buf, v.Len())
	reflect.Copy(reflect.ValueOf(e.buf), v)
	return e.write(e.buf)
}

func grow(b []byte, n int) []byte {
	if cap(b) >= n {
		return b[:n]
	}
	b = b[:cap(b)]
	b = append(b, make([]byte, n-len(b))...)
	return b
}

func (e *Encoder) EncodeBytesLen(l int) error {
	if l < 256 {
		return e.write1(codes.Bin8, uint8(l))
	}
	if l < 65536 {
		return e.write2(codes.Bin16, uint16(l))
	}
	return e.write4(codes.Bin32, uint32(l))
}

func (e *Encoder) encodeStrLen(l int) error {
	if l < 32 {
		return e.writeCode(codes.FixedStrLow | codes.Code(l))
	}
	if l < 256 {
		return e.write1(codes.Str8, uint8(l))
	}
	if l < 65536 {
		return e.write2(codes.Str16, uint16(l))
	}
	return e.write4(codes.Str32, uint32(l))
}

func (e *Encoder) EncodeString(v string) error {
	if err := e.encodeStrLen(len(v)); err != nil {
		return err
	}
	return e.writeString(v)
}

func (e *Encoder) EncodeBytes(v []byte) error {
	if v == nil {
		return e.EncodeNil()
	}
	if err := e.EncodeBytesLen(len(v)); err != nil {
		return err
	}
	return e.write(v)
}

func (e *Encoder) EncodeArrayLen(l int) error {
	if l < 16 {
		return e.writeCode(codes.FixedArrayLow | codes.Code(l))
	}
	if l < 65536 {
		return e.write2(codes.Array16, uint16(l))
	}
	return e.write4(codes.Array32, uint32(l))
}

func encodeStringSliceValue(e *Encoder, v reflect.Value) error {
	ss := v.Convert(sliceStringType).Interface().([]string)
	return e.encodeStringSlice(ss)
}

func (e *Encoder) encodeStringSlice(s []string) error {
	if s == nil {
		return e.EncodeNil()
	}
	if err := e.EncodeArrayLen(len(s)); err != nil {
		return err
	}
	for _, v := range s {
		if err := e.EncodeString(v); err != nil {
			return err
		}
	}
	return nil
}

func encodeSliceValue(e *Encoder, v reflect.Value) error {
	if v.IsNil() {
		return e.EncodeNil()
	}
	return encodeArrayValue(e, v)
}

func encodeArrayValue(e *Encoder, v reflect.Value) error {
	l := v.Len()
	if err := e.EncodeArrayLen(l); err != nil {
		return err
	}
	for i := 0; i < l; i++ {
		if err := e.EncodeValue(v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}
