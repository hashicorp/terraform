package msgpack

import (
	"encoding"
	"errors"
	"fmt"
	"reflect"
)

var interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
var stringType = reflect.TypeOf((*string)(nil)).Elem()

var valueDecoders []decoderFunc

//nolint:gochecknoinits
func init() {
	valueDecoders = []decoderFunc{
		reflect.Bool:          decodeBoolValue,
		reflect.Int:           decodeInt64Value,
		reflect.Int8:          decodeInt64Value,
		reflect.Int16:         decodeInt64Value,
		reflect.Int32:         decodeInt64Value,
		reflect.Int64:         decodeInt64Value,
		reflect.Uint:          decodeUint64Value,
		reflect.Uint8:         decodeUint64Value,
		reflect.Uint16:        decodeUint64Value,
		reflect.Uint32:        decodeUint64Value,
		reflect.Uint64:        decodeUint64Value,
		reflect.Float32:       decodeFloat32Value,
		reflect.Float64:       decodeFloat64Value,
		reflect.Complex64:     decodeUnsupportedValue,
		reflect.Complex128:    decodeUnsupportedValue,
		reflect.Array:         decodeArrayValue,
		reflect.Chan:          decodeUnsupportedValue,
		reflect.Func:          decodeUnsupportedValue,
		reflect.Interface:     decodeInterfaceValue,
		reflect.Map:           decodeMapValue,
		reflect.Ptr:           decodeUnsupportedValue,
		reflect.Slice:         decodeSliceValue,
		reflect.String:        decodeStringValue,
		reflect.Struct:        decodeStructValue,
		reflect.UnsafePointer: decodeUnsupportedValue,
	}
}

func mustSet(v reflect.Value) error {
	if !v.CanSet() {
		return fmt.Errorf("msgpack: Decode(nonsettable %s)", v.Type())
	}
	return nil
}

func getDecoder(typ reflect.Type) decoderFunc {
	if v, ok := typeDecMap.Load(typ); ok {
		return v.(decoderFunc)
	}
	fn := _getDecoder(typ)
	typeDecMap.Store(typ, fn)
	return fn
}

func _getDecoder(typ reflect.Type) decoderFunc {
	kind := typ.Kind()

	if typ.Implements(customDecoderType) {
		return decodeCustomValue
	}
	if typ.Implements(unmarshalerType) {
		return unmarshalValue
	}
	if typ.Implements(binaryUnmarshalerType) {
		return unmarshalBinaryValue
	}

	// Addressable struct field value.
	if kind != reflect.Ptr {
		ptr := reflect.PtrTo(typ)
		if ptr.Implements(customDecoderType) {
			return decodeCustomValueAddr
		}
		if ptr.Implements(unmarshalerType) {
			return unmarshalValueAddr
		}
		if ptr.Implements(binaryUnmarshalerType) {
			return unmarshalBinaryValueAddr
		}
	}

	switch kind {
	case reflect.Ptr:
		return ptrDecoderFunc(typ)
	case reflect.Slice:
		elem := typ.Elem()
		if elem.Kind() == reflect.Uint8 {
			return decodeBytesValue
		}
		if elem == stringType {
			return decodeStringSliceValue
		}
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return decodeByteArrayValue
		}
	case reflect.Map:
		if typ.Key() == stringType {
			switch typ.Elem() {
			case stringType:
				return decodeMapStringStringValue
			case interfaceType:
				return decodeMapStringInterfaceValue
			}
		}
	}

	return valueDecoders[kind]
}

func ptrDecoderFunc(typ reflect.Type) decoderFunc {
	decoder := getDecoder(typ.Elem())
	return func(d *Decoder, v reflect.Value) error {
		if d.hasNilCode() {
			if err := mustSet(v); err != nil {
				return err
			}
			if !v.IsNil() {
				v.Set(reflect.Zero(v.Type()))
			}
			return d.DecodeNil()
		}
		if v.IsNil() {
			if err := mustSet(v); err != nil {
				return err
			}
			v.Set(reflect.New(v.Type().Elem()))
		}
		return decoder(d, v.Elem())
	}
}

func decodeCustomValueAddr(d *Decoder, v reflect.Value) error {
	if !v.CanAddr() {
		return fmt.Errorf("msgpack: Decode(nonaddressable %T)", v.Interface())
	}
	return decodeCustomValue(d, v.Addr())
}

func decodeCustomValue(d *Decoder, v reflect.Value) error {
	if d.hasNilCode() {
		return d.decodeNilValue(v)
	}

	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	decoder := v.Interface().(CustomDecoder)
	return decoder.DecodeMsgpack(d)
}

func unmarshalValueAddr(d *Decoder, v reflect.Value) error {
	if !v.CanAddr() {
		return fmt.Errorf("msgpack: Decode(nonaddressable %T)", v.Interface())
	}
	return unmarshalValue(d, v.Addr())
}

func unmarshalValue(d *Decoder, v reflect.Value) error {
	if d.extLen == 0 || d.extLen == 1 {
		if d.hasNilCode() {
			return d.decodeNilValue(v)
		}
	}

	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	var b []byte

	if d.extLen != 0 {
		var err error
		b, err = d.readN(d.extLen)
		if err != nil {
			return err
		}
	} else {
		d.rec = make([]byte, 0, 64)
		if err := d.Skip(); err != nil {
			return err
		}
		b = d.rec
		d.rec = nil
	}

	unmarshaler := v.Interface().(Unmarshaler)
	return unmarshaler.UnmarshalMsgpack(b)
}

func decodeBoolValue(d *Decoder, v reflect.Value) error {
	flag, err := d.DecodeBool()
	if err != nil {
		return err
	}
	if err = mustSet(v); err != nil {
		return err
	}
	v.SetBool(flag)
	return nil
}

func decodeInterfaceValue(d *Decoder, v reflect.Value) error {
	if v.IsNil() {
		return d.interfaceValue(v)
	}

	elem := v.Elem()
	if !elem.CanAddr() {
		if d.hasNilCode() {
			v.Set(reflect.Zero(v.Type()))
			return d.DecodeNil()
		}
	}

	return d.DecodeValue(elem)
}

func (d *Decoder) interfaceValue(v reflect.Value) error {
	vv, err := d.decodeInterfaceCond()
	if err != nil {
		return err
	}

	if vv != nil {
		if v.Type() == errorType {
			if vv, ok := vv.(string); ok {
				v.Set(reflect.ValueOf(errors.New(vv)))
				return nil
			}
		}

		v.Set(reflect.ValueOf(vv))
	}

	return nil
}

func decodeUnsupportedValue(d *Decoder, v reflect.Value) error {
	return fmt.Errorf("msgpack: Decode(unsupported %s)", v.Type())
}

//------------------------------------------------------------------------------

func unmarshalBinaryValueAddr(d *Decoder, v reflect.Value) error {
	if !v.CanAddr() {
		return fmt.Errorf("msgpack: Decode(nonaddressable %T)", v.Interface())
	}
	return unmarshalBinaryValue(d, v.Addr())
}

func unmarshalBinaryValue(d *Decoder, v reflect.Value) error {
	if d.hasNilCode() {
		return d.decodeNilValue(v)
	}

	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	data, err := d.DecodeBytes()
	if err != nil {
		return err
	}

	unmarshaler := v.Interface().(encoding.BinaryUnmarshaler)
	return unmarshaler.UnmarshalBinary(data)
}
