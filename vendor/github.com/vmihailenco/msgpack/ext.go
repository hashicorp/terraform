package msgpack

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/vmihailenco/msgpack/codes"
)

var extTypes = make(map[int8]reflect.Type)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// RegisterExt records a type, identified by a value for that type,
// under the provided id. That id will identify the concrete type of a value
// sent or received as an interface variable. Only types that will be
// transferred as implementations of interface values need to be registered.
// Expecting to be used only during initialization, it panics if the mapping
// between types and ids is not a bijection.
func RegisterExt(id int8, value interface{}) {
	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	ptr := reflect.PtrTo(typ)

	if _, ok := extTypes[id]; ok {
		panic(fmt.Errorf("msgpack: ext with id=%d is already registered", id))
	}

	registerExt(id, ptr, getEncoder(ptr), nil)
	registerExt(id, typ, getEncoder(typ), getDecoder(typ))
}

func registerExt(id int8, typ reflect.Type, enc encoderFunc, dec decoderFunc) {
	if dec != nil {
		extTypes[id] = typ
	}
	if enc != nil {
		typEncMap[typ] = makeExtEncoder(id, enc)
	}
	if dec != nil {
		typDecMap[typ] = dec
	}
}

func (e *Encoder) EncodeExtHeader(typeId int8, length int) error {
	if err := e.encodeExtLen(length); err != nil {
		return err
	}
	if err := e.w.WriteByte(byte(typeId)); err != nil {
		return err
	}
	return nil
}

func makeExtEncoder(typeId int8, enc encoderFunc) encoderFunc {
	return func(e *Encoder, v reflect.Value) error {
		buf := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()

		oldw := e.w
		e.w = buf
		err := enc(e, v)
		e.w = oldw

		if err != nil {
			return err
		}

		if err := e.EncodeExtHeader(typeId, buf.Len()); err != nil {
			return err
		}
		return e.write(buf.Bytes())
	}
}

func (e *Encoder) encodeExtLen(l int) error {
	switch l {
	case 1:
		return e.writeCode(codes.FixExt1)
	case 2:
		return e.writeCode(codes.FixExt2)
	case 4:
		return e.writeCode(codes.FixExt4)
	case 8:
		return e.writeCode(codes.FixExt8)
	case 16:
		return e.writeCode(codes.FixExt16)
	}
	if l < 256 {
		return e.write1(codes.Ext8, uint64(l))
	}
	if l < 65536 {
		return e.write2(codes.Ext16, uint64(l))
	}
	return e.write4(codes.Ext32, uint32(l))
}

func (d *Decoder) parseExtLen(c codes.Code) (int, error) {
	switch c {
	case codes.FixExt1:
		return 1, nil
	case codes.FixExt2:
		return 2, nil
	case codes.FixExt4:
		return 4, nil
	case codes.FixExt8:
		return 8, nil
	case codes.FixExt16:
		return 16, nil
	case codes.Ext8:
		n, err := d.uint8()
		return int(n), err
	case codes.Ext16:
		n, err := d.uint16()
		return int(n), err
	case codes.Ext32:
		n, err := d.uint32()
		return int(n), err
	default:
		return 0, fmt.Errorf("msgpack: invalid code=%x decoding ext length", c)
	}
}

func (d *Decoder) decodeExtHeader(c codes.Code) (int8, int, error) {
	length, err := d.parseExtLen(c)
	if err != nil {
		return 0, 0, err
	}

	typeId, err := d.readCode()
	if err != nil {
		return 0, 0, err
	}

	return int8(typeId), length, nil
}

func (d *Decoder) DecodeExtHeader() (typeId int8, length int, err error) {
	c, err := d.readCode()
	if err != nil {
		return
	}
	return d.decodeExtHeader(c)
}

func (d *Decoder) extInterface(c codes.Code) (interface{}, error) {
	extId, extLen, err := d.decodeExtHeader(c)
	if err != nil {
		return nil, err
	}

	typ, ok := extTypes[extId]
	if !ok {
		return nil, fmt.Errorf("msgpack: unregistered ext id=%d", extId)
	}

	v := reflect.New(typ)

	d.extLen = extLen
	err = d.DecodeValue(v.Elem())
	d.extLen = 0
	if err != nil {
		return nil, err
	}

	return v.Interface(), nil
}

func (d *Decoder) skipExt(c codes.Code) error {
	n, err := d.parseExtLen(c)
	if err != nil {
		return err
	}
	return d.skipN(n + 1)
}

func (d *Decoder) skipExtHeader(c codes.Code) error {
	// Read ext type.
	_, err := d.readCode()
	if err != nil {
		return err
	}
	// Read ext body len.
	for i := 0; i < extHeaderLen(c); i++ {
		_, err := d.readCode()
		if err != nil {
			return err
		}
	}
	return nil
}

func extHeaderLen(c codes.Code) int {
	switch c {
	case codes.Ext8:
		return 1
	case codes.Ext16:
		return 2
	case codes.Ext32:
		return 4
	}
	return 0
}
