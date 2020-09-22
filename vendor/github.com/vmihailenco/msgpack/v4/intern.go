package msgpack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"

	"github.com/vmihailenco/msgpack/v4/codes"
)

var internStringExtID int8 = -128

var errUnexpectedCode = errors.New("msgpack: unexpected code")

func encodeInternInterfaceValue(e *Encoder, v reflect.Value) error {
	if v.IsNil() {
		return e.EncodeNil()
	}

	v = v.Elem()
	if v.Kind() == reflect.String {
		return encodeInternStringValue(e, v)
	}
	return e.EncodeValue(v)
}

func encodeInternStringValue(e *Encoder, v reflect.Value) error {
	s := v.String()

	if s != "" {
		if idx, ok := e.intern[s]; ok {
			return e.internStringIndex(idx)
		}

		if e.intern == nil {
			e.intern = make(map[string]int)
		}

		idx := len(e.intern)
		e.intern[s] = idx
	}

	return e.EncodeString(s)
}

func (e *Encoder) internStringIndex(idx int) error {
	if idx < math.MaxUint8 {
		if err := e.writeCode(codes.FixExt1); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(internStringExtID)); err != nil {
			return err
		}
		return e.w.WriteByte(byte(idx))
	}

	if idx < math.MaxUint16 {
		if err := e.writeCode(codes.FixExt2); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(internStringExtID)); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(idx >> 8)); err != nil {
			return err
		}
		return e.w.WriteByte(byte(idx))
	}

	if int64(idx) < math.MaxUint32 {
		if err := e.writeCode(codes.FixExt4); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(internStringExtID)); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(idx >> 24)); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(idx >> 16)); err != nil {
			return err
		}
		if err := e.w.WriteByte(byte(idx >> 8)); err != nil {
			return err
		}
		return e.w.WriteByte(byte(idx))
	}

	return fmt.Errorf("msgpack: intern string index=%d is too large", idx)
}

//------------------------------------------------------------------------------

func decodeInternInterfaceValue(d *Decoder, v reflect.Value) error {
	c, err := d.readCode()
	if err != nil {
		return err
	}

	s, err := d.internString(c)
	if err == nil {
		v.Set(reflect.ValueOf(s))
		return nil
	}
	if err != nil && err != errUnexpectedCode {
		return err
	}

	if err := d.s.UnreadByte(); err != nil {
		return err
	}

	return decodeInterfaceValue(d, v)
}

func decodeInternStringValue(d *Decoder, v reflect.Value) error {
	if err := mustSet(v); err != nil {
		return err
	}

	c, err := d.readCode()
	if err != nil {
		return err
	}

	s, err := d.internString(c)
	if err != nil {
		if err == errUnexpectedCode {
			return fmt.Errorf("msgpack: invalid code=%x decoding intern string", c)
		}
		return err
	}

	v.SetString(s)
	return nil
}

func (d *Decoder) internString(c codes.Code) (string, error) {
	if codes.IsFixedString(c) {
		n := int(c & codes.FixedStrMask)
		return d.internStringWithLen(n)
	}

	switch c {
	case codes.FixExt1, codes.FixExt2, codes.FixExt4:
		typeID, length, err := d.extHeader(c)
		if err != nil {
			return "", err
		}
		if typeID != internStringExtID {
			err := fmt.Errorf("msgpack: got ext type=%d, wanted %d",
				typeID, internStringExtID)
			return "", err
		}

		idx, err := d.internStringIndex(length)
		if err != nil {
			return "", err
		}

		return d.internStringAtIndex(idx)
	case codes.Str8, codes.Bin8:
		n, err := d.uint8()
		if err != nil {
			return "", err
		}
		return d.internStringWithLen(int(n))
	case codes.Str16, codes.Bin16:
		n, err := d.uint16()
		if err != nil {
			return "", err
		}
		return d.internStringWithLen(int(n))
	case codes.Str32, codes.Bin32:
		n, err := d.uint32()
		if err != nil {
			return "", err
		}
		return d.internStringWithLen(int(n))
	}

	return "", errUnexpectedCode
}

func (d *Decoder) internStringIndex(length int) (int, error) {
	switch length {
	case 1:
		c, err := d.s.ReadByte()
		if err != nil {
			return 0, err
		}
		return int(c), nil
	case 2:
		b, err := d.readN(2)
		if err != nil {
			return 0, err
		}
		n := binary.BigEndian.Uint16(b)
		return int(n), nil
	case 4:
		b, err := d.readN(4)
		if err != nil {
			return 0, err
		}
		n := binary.BigEndian.Uint32(b)
		return int(n), nil
	}

	err := fmt.Errorf("msgpack: unsupported intern string index length=%d", length)
	return 0, err
}

func (d *Decoder) internStringAtIndex(idx int) (string, error) {
	if idx >= len(d.intern) {
		err := fmt.Errorf("msgpack: intern string with index=%d does not exist", idx)
		return "", err
	}
	return d.intern[idx], nil
}

func (d *Decoder) internStringWithLen(n int) (string, error) {
	if n <= 0 {
		return "", nil
	}

	s, err := d.stringWithLen(n)
	if err != nil {
		return "", err
	}

	d.intern = append(d.intern, s)

	return s, nil
}
