package msgpack

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/vmihailenco/msgpack/v4/codes"
)

var (
	mapStringStringPtrType = reflect.TypeOf((*map[string]string)(nil))
	mapStringStringType    = mapStringStringPtrType.Elem()
)

var (
	mapStringInterfacePtrType = reflect.TypeOf((*map[string]interface{})(nil))
	mapStringInterfaceType    = mapStringInterfacePtrType.Elem()
)

func decodeMapValue(d *Decoder, v reflect.Value) error {
	size, err := d.DecodeMapLen()
	if err != nil {
		return err
	}

	typ := v.Type()
	if size == -1 {
		v.Set(reflect.Zero(typ))
		return nil
	}

	if v.IsNil() {
		v.Set(reflect.MakeMap(typ))
	}
	if size == 0 {
		return nil
	}

	return decodeMapValueSize(d, v, size)
}

func decodeMapValueSize(d *Decoder, v reflect.Value, size int) error {
	typ := v.Type()
	keyType := typ.Key()
	valueType := typ.Elem()

	for i := 0; i < size; i++ {
		mk := reflect.New(keyType).Elem()
		if err := d.DecodeValue(mk); err != nil {
			return err
		}

		mv := reflect.New(valueType).Elem()
		if err := d.DecodeValue(mv); err != nil {
			return err
		}

		v.SetMapIndex(mk, mv)
	}

	return nil
}

// DecodeMapLen decodes map length. Length is -1 when map is nil.
func (d *Decoder) DecodeMapLen() (int, error) {
	c, err := d.readCode()
	if err != nil {
		return 0, err
	}

	if codes.IsExt(c) {
		if err = d.skipExtHeader(c); err != nil {
			return 0, err
		}

		c, err = d.readCode()
		if err != nil {
			return 0, err
		}
	}
	return d.mapLen(c)
}

func (d *Decoder) mapLen(c codes.Code) (int, error) {
	size, err := d._mapLen(c)
	err = expandInvalidCodeMapLenError(c, err)
	return size, err
}

func (d *Decoder) _mapLen(c codes.Code) (int, error) {
	if c == codes.Nil {
		return -1, nil
	}
	if c >= codes.FixedMapLow && c <= codes.FixedMapHigh {
		return int(c & codes.FixedMapMask), nil
	}
	if c == codes.Map16 {
		size, err := d.uint16()
		return int(size), err
	}
	if c == codes.Map32 {
		size, err := d.uint32()
		return int(size), err
	}
	return 0, errInvalidCode
}

var errInvalidCode = errors.New("invalid code")

func expandInvalidCodeMapLenError(c codes.Code, err error) error {
	if err == errInvalidCode {
		return fmt.Errorf("msgpack: invalid code=%x decoding map length", c)
	}
	return err
}

func decodeMapStringStringValue(d *Decoder, v reflect.Value) error {
	mptr := v.Addr().Convert(mapStringStringPtrType).Interface().(*map[string]string)
	return d.decodeMapStringStringPtr(mptr)
}

func (d *Decoder) decodeMapStringStringPtr(ptr *map[string]string) error {
	size, err := d.DecodeMapLen()
	if err != nil {
		return err
	}
	if size == -1 {
		*ptr = nil
		return nil
	}

	m := *ptr
	if m == nil {
		*ptr = make(map[string]string, min(size, maxMapSize))
		m = *ptr
	}

	for i := 0; i < size; i++ {
		mk, err := d.DecodeString()
		if err != nil {
			return err
		}
		mv, err := d.DecodeString()
		if err != nil {
			return err
		}
		m[mk] = mv
	}

	return nil
}

func decodeMapStringInterfaceValue(d *Decoder, v reflect.Value) error {
	ptr := v.Addr().Convert(mapStringInterfacePtrType).Interface().(*map[string]interface{})
	return d.decodeMapStringInterfacePtr(ptr)
}

func (d *Decoder) decodeMapStringInterfacePtr(ptr *map[string]interface{}) error {
	n, err := d.DecodeMapLen()
	if err != nil {
		return err
	}
	if n == -1 {
		*ptr = nil
		return nil
	}

	m := *ptr
	if m == nil {
		*ptr = make(map[string]interface{}, min(n, maxMapSize))
		m = *ptr
	}

	for i := 0; i < n; i++ {
		mk, err := d.DecodeString()
		if err != nil {
			return err
		}
		mv, err := d.decodeInterfaceCond()
		if err != nil {
			return err
		}
		m[mk] = mv
	}

	return nil
}

var errUnsupportedMapKey = errors.New("msgpack: unsupported map key")

func (d *Decoder) DecodeMap() (interface{}, error) {
	if d.decodeMapFunc != nil {
		return d.decodeMapFunc(d)
	}

	size, err := d.DecodeMapLen()
	if err != nil {
		return nil, err
	}
	if size == -1 {
		return nil, nil
	}
	if size == 0 {
		return make(map[string]interface{}), nil
	}

	code, err := d.PeekCode()
	if err != nil {
		return nil, err
	}

	if codes.IsString(code) || codes.IsBin(code) {
		return d.decodeMapStringInterfaceSize(size)
	}

	key, err := d.decodeInterfaceCond()
	if err != nil {
		return nil, err
	}

	value, err := d.decodeInterfaceCond()
	if err != nil {
		return nil, err
	}

	keyType := reflect.TypeOf(key)
	valueType := reflect.TypeOf(value)

	if !keyType.Comparable() {
		return nil, errUnsupportedMapKey
	}

	mapType := reflect.MapOf(keyType, valueType)
	mapValue := reflect.MakeMap(mapType)

	mapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	size--

	err = decodeMapValueSize(d, mapValue, size)
	if err != nil {
		return nil, err
	}

	return mapValue.Interface(), nil
}

func (d *Decoder) decodeMapStringInterfaceSize(size int) (map[string]interface{}, error) {
	m := make(map[string]interface{}, min(size, maxMapSize))
	for i := 0; i < size; i++ {
		mk, err := d.DecodeString()
		if err != nil {
			return nil, err
		}
		mv, err := d.decodeInterfaceCond()
		if err != nil {
			return nil, err
		}
		m[mk] = mv
	}
	return m, nil
}

func (d *Decoder) skipMap(c codes.Code) error {
	n, err := d.mapLen(c)
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if err := d.Skip(); err != nil {
			return err
		}
		if err := d.Skip(); err != nil {
			return err
		}
	}
	return nil
}

func decodeStructValue(d *Decoder, v reflect.Value) error {
	c, err := d.readCode()
	if err != nil {
		return err
	}

	var isArray bool

	n, err := d._mapLen(c)
	if err != nil {
		var err2 error
		n, err2 = d.arrayLen(c)
		if err2 != nil {
			return expandInvalidCodeMapLenError(c, err)
		}
		isArray = true
	}
	if n == -1 {
		if err = mustSet(v); err != nil {
			return err
		}
		v.Set(reflect.Zero(v.Type()))
		return nil
	}

	var fields *fields
	if d.flags&decodeUsingJSONFlag != 0 {
		fields = jsonStructs.Fields(v.Type())
	} else {
		fields = structs.Fields(v.Type())
	}

	if isArray {
		for i, f := range fields.List {
			if i >= n {
				break
			}
			if err := f.DecodeValue(d, v); err != nil {
				return err
			}
		}

		// Skip extra values.
		for i := len(fields.List); i < n; i++ {
			if err := d.Skip(); err != nil {
				return err
			}
		}

		return nil
	}

	for i := 0; i < n; i++ {
		name, err := d.DecodeString()
		if err != nil {
			return err
		}

		if f := fields.Map[name]; f != nil {
			if err := f.DecodeValue(d, v); err != nil {
				return err
			}
		} else if d.flags&disallowUnknownFieldsFlag != 0 {
			return fmt.Errorf("msgpack: unknown field %q", name)
		} else if err := d.Skip(); err != nil {
			return err
		}
	}

	return nil
}
