package msgpack

import (
	"fmt"
	"reflect"
)

var valueEncoders []encoderFunc

func init() {
	valueEncoders = []encoderFunc{
		reflect.Bool:          encodeBoolValue,
		reflect.Int:           encodeInt64CondValue,
		reflect.Int8:          encodeInt8CondValue,
		reflect.Int16:         encodeInt16CondValue,
		reflect.Int32:         encodeInt32CondValue,
		reflect.Int64:         encodeInt64CondValue,
		reflect.Uint:          encodeUint64CondValue,
		reflect.Uint8:         encodeUint8CondValue,
		reflect.Uint16:        encodeUint16CondValue,
		reflect.Uint32:        encodeUint32CondValue,
		reflect.Uint64:        encodeUint64CondValue,
		reflect.Float32:       encodeFloat32Value,
		reflect.Float64:       encodeFloat64Value,
		reflect.Complex64:     encodeUnsupportedValue,
		reflect.Complex128:    encodeUnsupportedValue,
		reflect.Array:         encodeArrayValue,
		reflect.Chan:          encodeUnsupportedValue,
		reflect.Func:          encodeUnsupportedValue,
		reflect.Interface:     encodeInterfaceValue,
		reflect.Map:           encodeMapValue,
		reflect.Ptr:           encodeUnsupportedValue,
		reflect.Slice:         encodeSliceValue,
		reflect.String:        encodeStringValue,
		reflect.Struct:        encodeStructValue,
		reflect.UnsafePointer: encodeUnsupportedValue,
	}
}

func getEncoder(typ reflect.Type) encoderFunc {
	if encoder, ok := typEncMap[typ]; ok {
		return encoder
	}

	if typ.Implements(customEncoderType) {
		return encodeCustomValue
	}
	if typ.Implements(marshalerType) {
		return marshalValue
	}

	kind := typ.Kind()

	// Addressable struct field value.
	if kind != reflect.Ptr {
		ptr := reflect.PtrTo(typ)
		if ptr.Implements(customEncoderType) {
			return encodeCustomValuePtr
		}
		if ptr.Implements(marshalerType) {
			return marshalValuePtr
		}
	}

	if typ == errorType {
		return encodeErrorValue
	}

	switch kind {
	case reflect.Ptr:
		return ptrEncoderFunc(typ)
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return encodeByteSliceValue
		}
	case reflect.Array:
		if typ.Elem().Kind() == reflect.Uint8 {
			return encodeByteArrayValue
		}
	case reflect.Map:
		if typ.Key() == stringType {
			switch typ.Elem() {
			case stringType:
				return encodeMapStringStringValue
			case interfaceType:
				return encodeMapStringInterfaceValue
			}
		}
	}
	return valueEncoders[kind]
}

func ptrEncoderFunc(typ reflect.Type) encoderFunc {
	encoder := getEncoder(typ.Elem())
	return func(e *Encoder, v reflect.Value) error {
		if v.IsNil() {
			return e.EncodeNil()
		}
		return encoder(e, v.Elem())
	}
}

func encodeCustomValuePtr(e *Encoder, v reflect.Value) error {
	if !v.CanAddr() {
		return fmt.Errorf("msgpack: Encode(non-addressable %T)", v.Interface())
	}
	encoder := v.Addr().Interface().(CustomEncoder)
	return encoder.EncodeMsgpack(e)
}

func encodeCustomValue(e *Encoder, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if v.IsNil() {
			return e.EncodeNil()
		}
	}

	encoder := v.Interface().(CustomEncoder)
	return encoder.EncodeMsgpack(e)
}

func marshalValuePtr(e *Encoder, v reflect.Value) error {
	if !v.CanAddr() {
		return fmt.Errorf("msgpack: Encode(non-addressable %T)", v.Interface())
	}
	return marshalValue(e, v.Addr())
}

func marshalValue(e *Encoder, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if v.IsNil() {
			return e.EncodeNil()
		}
	}

	marshaler := v.Interface().(Marshaler)
	b, err := marshaler.MarshalMsgpack()
	if err != nil {
		return err
	}
	_, err = e.w.Write(b)
	return err
}

func encodeBoolValue(e *Encoder, v reflect.Value) error {
	return e.EncodeBool(v.Bool())
}

func encodeInterfaceValue(e *Encoder, v reflect.Value) error {
	if v.IsNil() {
		return e.EncodeNil()
	}
	return e.EncodeValue(v.Elem())
}

func encodeErrorValue(e *Encoder, v reflect.Value) error {
	if v.IsNil() {
		return e.EncodeNil()
	}
	return e.EncodeString(v.Interface().(error).Error())
}

func encodeUnsupportedValue(e *Encoder, v reflect.Value) error {
	return fmt.Errorf("msgpack: Encode(unsupported %s)", v.Type())
}
