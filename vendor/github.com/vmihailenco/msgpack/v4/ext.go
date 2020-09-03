package msgpack

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"

	"github.com/vmihailenco/msgpack/v4/codes"
)

type extInfo struct {
	Type    reflect.Type
	Decoder decoderFunc
}

var extTypes = make(map[int8]*extInfo)

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

	registerExt(id, ptr, getEncoder(ptr), getDecoder(ptr))
	registerExt(id, typ, getEncoder(typ), getDecoder(typ))
}

func registerExt(id int8, typ reflect.Type, enc encoderFunc, dec decoderFunc) {
	if enc != nil {
		typeEncMap.Store(typ, makeExtEncoder(id, enc))
	}
	if dec != nil {
		extTypes[id] = &extInfo{
			Type:    typ,
			Decoder: dec,
		}
		typeDecMap.Store(typ, makeExtDecoder(id, dec))
	}
}

func (e *Encoder) EncodeExtHeader(typeID int8, length int) error {
	if err := e.encodeExtLen(length); err != nil {
		return err
	}
	if err := e.w.WriteByte(byte(typeID)); err != nil {
		return err
	}
	return nil
}

func makeExtEncoder(typeID int8, enc encoderFunc) encoderFunc {
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

		err = e.EncodeExtHeader(typeID, buf.Len())
		if err != nil {
			return err
		}
		return e.write(buf.Bytes())
	}
}

func makeExtDecoder(typeID int8, dec decoderFunc) decoderFunc {
	return func(d *Decoder, v reflect.Value) error {
		c, err := d.PeekCode()
		if err != nil {
			return err
		}

		if !codes.IsExt(c) {
			return dec(d, v)
		}

		id, extLen, err := d.DecodeExtHeader()
		if err != nil {
			return err
		}

		if id != typeID {
			return fmt.Errorf("msgpack: got ext type=%d, wanted %d", id, typeID)
		}

		d.extLen = extLen
		return dec(d, v)
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
		return e.write1(codes.Ext8, uint8(l))
	}
	if l < 65536 {
		return e.write2(codes.Ext16, uint16(l))
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

func (d *Decoder) extHeader(c codes.Code) (int8, int, error) {
	length, err := d.parseExtLen(c)
	if err != nil {
		return 0, 0, err
	}

	typeID, err := d.readCode()
	if err != nil {
		return 0, 0, err
	}

	return int8(typeID), length, nil
}

func (d *Decoder) DecodeExtHeader() (typeID int8, length int, err error) {
	c, err := d.readCode()
	if err != nil {
		return
	}
	return d.extHeader(c)
}

func (d *Decoder) extInterface(c codes.Code) (interface{}, error) {
	extID, extLen, err := d.extHeader(c)
	if err != nil {
		return nil, err
	}

	info, ok := extTypes[extID]
	if !ok {
		return nil, fmt.Errorf("msgpack: unknown ext id=%d", extID)
	}

	v := reflect.New(info.Type)

	d.extLen = extLen
	err = info.Decoder(d, v.Elem())
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
