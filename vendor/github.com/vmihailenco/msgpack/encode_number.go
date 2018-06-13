package msgpack

import (
	"math"
	"reflect"

	"github.com/vmihailenco/msgpack/codes"
)

// EncodeUint encodes an uint64 in 1, 2, 3, 5, or 9 bytes.
func (e *Encoder) EncodeUint(v uint64) error {
	if v <= math.MaxInt8 {
		return e.w.WriteByte(byte(v))
	}
	if v <= math.MaxUint8 {
		return e.write1(codes.Uint8, v)
	}
	if v <= math.MaxUint16 {
		return e.write2(codes.Uint16, v)
	}
	if v <= math.MaxUint32 {
		return e.write4(codes.Uint32, uint32(v))
	}
	return e.write8(codes.Uint64, v)
}

// EncodeInt encodes an int64 in 1, 2, 3, 5, or 9 bytes.
func (e *Encoder) EncodeInt(v int64) error {
	if v >= 0 {
		return e.EncodeUint(uint64(v))
	}
	if v >= int64(int8(codes.NegFixedNumLow)) {
		return e.w.WriteByte(byte(v))
	}
	if v >= math.MinInt8 {
		return e.write1(codes.Int8, uint64(v))
	}
	if v >= math.MinInt16 {
		return e.write2(codes.Int16, uint64(v))
	}
	if v >= math.MinInt32 {
		return e.write4(codes.Int32, uint32(v))
	}
	return e.write8(codes.Int64, uint64(v))
}

func (e *Encoder) EncodeFloat32(n float32) error {
	return e.write4(codes.Float, math.Float32bits(n))
}

func (e *Encoder) EncodeFloat64(n float64) error {
	return e.write8(codes.Double, math.Float64bits(n))
}

func (e *Encoder) write1(code codes.Code, n uint64) error {
	e.buf = e.buf[:2]
	e.buf[0] = byte(code)
	e.buf[1] = byte(n)
	return e.write(e.buf)
}

func (e *Encoder) write2(code codes.Code, n uint64) error {
	e.buf = e.buf[:3]
	e.buf[0] = byte(code)
	e.buf[1] = byte(n >> 8)
	e.buf[2] = byte(n)
	return e.write(e.buf)
}

func (e *Encoder) write4(code codes.Code, n uint32) error {
	e.buf = e.buf[:5]
	e.buf[0] = byte(code)
	e.buf[1] = byte(n >> 24)
	e.buf[2] = byte(n >> 16)
	e.buf[3] = byte(n >> 8)
	e.buf[4] = byte(n)
	return e.write(e.buf)
}

func (e *Encoder) write8(code codes.Code, n uint64) error {
	e.buf = e.buf[:9]
	e.buf[0] = byte(code)
	e.buf[1] = byte(n >> 56)
	e.buf[2] = byte(n >> 48)
	e.buf[3] = byte(n >> 40)
	e.buf[4] = byte(n >> 32)
	e.buf[5] = byte(n >> 24)
	e.buf[6] = byte(n >> 16)
	e.buf[7] = byte(n >> 8)
	e.buf[8] = byte(n)
	return e.write(e.buf)
}

func encodeInt64Value(e *Encoder, v reflect.Value) error {
	return e.EncodeInt(v.Int())
}

func encodeUint64Value(e *Encoder, v reflect.Value) error {
	return e.EncodeUint(v.Uint())
}

func encodeFloat32Value(e *Encoder, v reflect.Value) error {
	return e.EncodeFloat32(float32(v.Float()))
}

func encodeFloat64Value(e *Encoder, v reflect.Value) error {
	return e.EncodeFloat64(v.Float())
}
