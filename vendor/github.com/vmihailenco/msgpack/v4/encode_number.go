package msgpack

import (
	"math"
	"reflect"

	"github.com/vmihailenco/msgpack/v4/codes"
)

// EncodeUint8 encodes an uint8 in 2 bytes preserving type of the number.
func (e *Encoder) EncodeUint8(n uint8) error {
	return e.write1(codes.Uint8, n)
}

func (e *Encoder) encodeUint8Cond(n uint8) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeUint(uint64(n))
	}
	return e.EncodeUint8(n)
}

// EncodeUint16 encodes an uint16 in 3 bytes preserving type of the number.
func (e *Encoder) EncodeUint16(n uint16) error {
	return e.write2(codes.Uint16, n)
}

func (e *Encoder) encodeUint16Cond(n uint16) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeUint(uint64(n))
	}
	return e.EncodeUint16(n)
}

// EncodeUint32 encodes an uint16 in 5 bytes preserving type of the number.
func (e *Encoder) EncodeUint32(n uint32) error {
	return e.write4(codes.Uint32, n)
}

func (e *Encoder) encodeUint32Cond(n uint32) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeUint(uint64(n))
	}
	return e.EncodeUint32(n)
}

// EncodeUint64 encodes an uint16 in 9 bytes preserving type of the number.
func (e *Encoder) EncodeUint64(n uint64) error {
	return e.write8(codes.Uint64, n)
}

func (e *Encoder) encodeUint64Cond(n uint64) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeUint(n)
	}
	return e.EncodeUint64(n)
}

// EncodeInt8 encodes an int8 in 2 bytes preserving type of the number.
func (e *Encoder) EncodeInt8(n int8) error {
	return e.write1(codes.Int8, uint8(n))
}

func (e *Encoder) encodeInt8Cond(n int8) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeInt(int64(n))
	}
	return e.EncodeInt8(n)
}

// EncodeInt16 encodes an int16 in 3 bytes preserving type of the number.
func (e *Encoder) EncodeInt16(n int16) error {
	return e.write2(codes.Int16, uint16(n))
}

func (e *Encoder) encodeInt16Cond(n int16) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeInt(int64(n))
	}
	return e.EncodeInt16(n)
}

// EncodeInt32 encodes an int32 in 5 bytes preserving type of the number.
func (e *Encoder) EncodeInt32(n int32) error {
	return e.write4(codes.Int32, uint32(n))
}

func (e *Encoder) encodeInt32Cond(n int32) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeInt(int64(n))
	}
	return e.EncodeInt32(n)
}

// EncodeInt64 encodes an int64 in 9 bytes preserving type of the number.
func (e *Encoder) EncodeInt64(n int64) error {
	return e.write8(codes.Int64, uint64(n))
}

func (e *Encoder) encodeInt64Cond(n int64) error {
	if e.flags&useCompactIntsFlag != 0 {
		return e.EncodeInt(n)
	}
	return e.EncodeInt64(n)
}

// EncodeUnsignedNumber encodes an uint64 in 1, 2, 3, 5, or 9 bytes.
// Type of the number is lost during encoding.
func (e *Encoder) EncodeUint(n uint64) error {
	if n <= math.MaxInt8 {
		return e.w.WriteByte(byte(n))
	}
	if n <= math.MaxUint8 {
		return e.EncodeUint8(uint8(n))
	}
	if n <= math.MaxUint16 {
		return e.EncodeUint16(uint16(n))
	}
	if n <= math.MaxUint32 {
		return e.EncodeUint32(uint32(n))
	}
	return e.EncodeUint64(n)
}

// EncodeNumber encodes an int64 in 1, 2, 3, 5, or 9 bytes.
// Type of the number is lost during encoding.
func (e *Encoder) EncodeInt(n int64) error {
	if n >= 0 {
		return e.EncodeUint(uint64(n))
	}
	if n >= int64(int8(codes.NegFixedNumLow)) {
		return e.w.WriteByte(byte(n))
	}
	if n >= math.MinInt8 {
		return e.EncodeInt8(int8(n))
	}
	if n >= math.MinInt16 {
		return e.EncodeInt16(int16(n))
	}
	if n >= math.MinInt32 {
		return e.EncodeInt32(int32(n))
	}
	return e.EncodeInt64(n)
}

func (e *Encoder) EncodeFloat32(n float32) error {
	if e.flags&useCompactFloatsFlag != 0 {
		if float32(int64(n)) == n {
			return e.EncodeInt(int64(n))
		}
	}
	return e.write4(codes.Float, math.Float32bits(n))
}

func (e *Encoder) EncodeFloat64(n float64) error {
	if e.flags&useCompactFloatsFlag != 0 {
		// Both NaN and Inf convert to int64(-0x8000000000000000)
		// If n is NaN then it never compares true with any other value
		// If n is Inf then it doesn't convert from int64 back to +/-Inf
		// In both cases the comparison works.
		if float64(int64(n)) == n {
			return e.EncodeInt(int64(n))
		}
	}
	return e.write8(codes.Double, math.Float64bits(n))
}

func (e *Encoder) write1(code codes.Code, n uint8) error {
	e.buf = e.buf[:2]
	e.buf[0] = byte(code)
	e.buf[1] = n
	return e.write(e.buf)
}

func (e *Encoder) write2(code codes.Code, n uint16) error {
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

func encodeUint8CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeUint8Cond(uint8(v.Uint()))
}

func encodeUint16CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeUint16Cond(uint16(v.Uint()))
}

func encodeUint32CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeUint32Cond(uint32(v.Uint()))
}

func encodeUint64CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeUint64Cond(v.Uint())
}

func encodeInt8CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeInt8Cond(int8(v.Int()))
}

func encodeInt16CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeInt16Cond(int16(v.Int()))
}

func encodeInt32CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeInt32Cond(int32(v.Int()))
}

func encodeInt64CondValue(e *Encoder, v reflect.Value) error {
	return e.encodeInt64Cond(v.Int())
}

func encodeFloat32Value(e *Encoder, v reflect.Value) error {
	return e.EncodeFloat32(float32(v.Float()))
}

func encodeFloat64Value(e *Encoder, v reflect.Value) error {
	return e.EncodeFloat64(v.Float())
}
