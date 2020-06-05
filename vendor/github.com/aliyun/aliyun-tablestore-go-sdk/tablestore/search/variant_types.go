package search

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
)

type VariantValue []byte
type VariantType byte

const (
	// variant type
	VT_INTEGER VariantType = 0x0
	VT_DOUBLE  VariantType = 0x1
	VT_BOOLEAN VariantType = 0x2
	VT_STRING  VariantType = 0x3
)

func ToVariantValue(value interface{}) (VariantValue, error) {
	t := reflect.TypeOf(value)
	switch t.Kind() {
	case reflect.String:
		return VTString(value.(string)), nil
	case reflect.Int:
		return VTInteger(int64(value.(int))), nil
	case reflect.Int64:
		return VTInteger(value.(int64)), nil
	case reflect.Float64:
		return VTDouble(value.(float64)), nil
	case reflect.Bool:
		return VTBoolean(value.(bool)), nil
	default:
		return nil, errors.New("interface{} type must be string/int64/float64.")
	}
}

func (v *VariantValue) GetType() VariantType {
	return VariantType(([]byte)(*v)[0])
}

func VTInteger(v int64) VariantValue {
	buf := make([]byte, 9)
	buf[0] = byte(VT_INTEGER)
	binary.LittleEndian.PutUint64(buf[1:9], uint64(v))
	return (VariantValue)(buf)
}

func VTDouble(v float64) VariantValue {
	buf := make([]byte, 9)
	buf[0] = byte(VT_DOUBLE)
	binary.LittleEndian.PutUint64(buf[1:9], math.Float64bits(v))
	return (VariantValue)(buf)
}

func VTString(v string) VariantValue {
	buf := make([]byte, 5+len(v))
	buf[0] = byte(VT_STRING)
	binary.LittleEndian.PutUint32(buf[1:5], uint32(len(v)))
	copy(buf[5:], v)
	return (VariantValue)(buf)
}

func VTBoolean(b bool) VariantValue {
	buf := make([]byte, 2)
	buf[0] = byte(VT_BOOLEAN)
	if b {
		buf[1] = 1
	} else {
		buf[1] = 0
	}
	return (VariantValue)(buf)
}
