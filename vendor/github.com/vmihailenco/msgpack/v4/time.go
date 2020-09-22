package msgpack

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"

	"github.com/vmihailenco/msgpack/v4/codes"
)

var timeExtID int8 = -1

var timePtrType = reflect.TypeOf((*time.Time)(nil))

//nolint:gochecknoinits
func init() {
	registerExt(timeExtID, timePtrType.Elem(), encodeTimeValue, decodeTimeValue)
}

func (e *Encoder) EncodeTime(tm time.Time) error {
	b := e.encodeTime(tm)
	if err := e.encodeExtLen(len(b)); err != nil {
		return err
	}
	if err := e.w.WriteByte(byte(timeExtID)); err != nil {
		return err
	}
	return e.write(b)
}

func (e *Encoder) encodeTime(tm time.Time) []byte {
	secs := uint64(tm.Unix())
	if secs>>34 == 0 {
		data := uint64(tm.Nanosecond())<<34 | secs
		if data&0xffffffff00000000 == 0 {
			b := e.timeBuf[:4]
			binary.BigEndian.PutUint32(b, uint32(data))
			return b
		}
		b := e.timeBuf[:8]
		binary.BigEndian.PutUint64(b, data)
		return b
	}

	b := e.timeBuf[:12]
	binary.BigEndian.PutUint32(b, uint32(tm.Nanosecond()))
	binary.BigEndian.PutUint64(b[4:], secs)
	return b
}

func (d *Decoder) DecodeTime() (time.Time, error) {
	tm, err := d.decodeTime()
	if err != nil {
		return tm, err
	}

	if tm.IsZero() {
		// Assume that zero time does not have timezone information.
		return tm.UTC(), nil
	}
	return tm, nil
}

func (d *Decoder) decodeTime() (time.Time, error) {
	extLen := d.extLen
	d.extLen = 0
	if extLen == 0 {
		c, err := d.readCode()
		if err != nil {
			return time.Time{}, err
		}

		// Legacy format.
		if c == codes.FixedArrayLow|2 {
			sec, err := d.DecodeInt64()
			if err != nil {
				return time.Time{}, err
			}

			nsec, err := d.DecodeInt64()
			if err != nil {
				return time.Time{}, err
			}

			return time.Unix(sec, nsec), nil
		}

		if codes.IsString(c) {
			s, err := d.string(c)
			if err != nil {
				return time.Time{}, err
			}
			return time.Parse(time.RFC3339Nano, s)
		}

		extLen, err = d.parseExtLen(c)
		if err != nil {
			return time.Time{}, err
		}

		// Skip ext id.
		_, err = d.s.ReadByte()
		if err != nil {
			return time.Time{}, nil
		}
	}

	b, err := d.readN(extLen)
	if err != nil {
		return time.Time{}, err
	}

	switch len(b) {
	case 4:
		sec := binary.BigEndian.Uint32(b)
		return time.Unix(int64(sec), 0), nil
	case 8:
		sec := binary.BigEndian.Uint64(b)
		nsec := int64(sec >> 34)
		sec &= 0x00000003ffffffff
		return time.Unix(int64(sec), nsec), nil
	case 12:
		nsec := binary.BigEndian.Uint32(b)
		sec := binary.BigEndian.Uint64(b[4:])
		return time.Unix(int64(sec), int64(nsec)), nil
	default:
		err = fmt.Errorf("msgpack: invalid ext len=%d decoding time", extLen)
		return time.Time{}, err
	}
}

func encodeTimeValue(e *Encoder, v reflect.Value) error {
	tm := v.Interface().(time.Time)
	b := e.encodeTime(tm)
	return e.write(b)
}

func decodeTimeValue(d *Decoder, v reflect.Value) error {
	tm, err := d.DecodeTime()
	if err != nil {
		return err
	}

	ptr := v.Addr().Interface().(*time.Time)
	*ptr = tm

	return nil
}
