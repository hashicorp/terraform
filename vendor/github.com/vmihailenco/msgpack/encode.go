package msgpack

import (
	"bytes"
	"io"
	"reflect"
	"time"

	"github.com/vmihailenco/msgpack/codes"
)

type writer interface {
	io.Writer
	WriteByte(byte) error
	WriteString(string) (int, error)
}

type byteWriter struct {
	io.Writer

	buf       []byte
	bootstrap [64]byte
}

func newByteWriter(w io.Writer) *byteWriter {
	bw := &byteWriter{
		Writer: w,
	}
	bw.buf = bw.bootstrap[:]
	return bw
}

func (w *byteWriter) WriteByte(c byte) error {
	w.buf = w.buf[:1]
	w.buf[0] = c
	_, err := w.Write(w.buf)
	return err
}

func (w *byteWriter) WriteString(s string) (int, error) {
	w.buf = append(w.buf[:0], s...)
	return w.Write(w.buf)
}

// Marshal returns the MessagePack encoding of v.
func Marshal(v ...interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := NewEncoder(&buf).Encode(v...)
	return buf.Bytes(), err
}

type Encoder struct {
	w   writer
	buf []byte

	sortMapKeys   bool
	structAsArray bool
	useJSONTag    bool
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	bw, ok := w.(writer)
	if !ok {
		bw = newByteWriter(w)
	}
	return &Encoder{
		w:   bw,
		buf: make([]byte, 9),
	}
}

// SortMapKeys causes the Encoder to encode map keys in increasing order.
// Supported map types are:
//   - map[string]string
//   - map[string]interface{}
func (e *Encoder) SortMapKeys(v bool) *Encoder {
	e.sortMapKeys = v
	return e
}

// StructAsArray causes the Encoder to encode Go structs as MessagePack arrays.
func (e *Encoder) StructAsArray(v bool) *Encoder {
	e.structAsArray = v
	return e
}

// UseJSONTag causes the Encoder to use json struct tag as fallback option
// if there is no msgpack tag.
func (e *Encoder) UseJSONTag(v bool) *Encoder {
	e.useJSONTag = v
	return e
}

func (e *Encoder) Encode(v ...interface{}) error {
	for _, vv := range v {
		if err := e.encode(vv); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encode(v interface{}) error {
	switch v := v.(type) {
	case nil:
		return e.EncodeNil()
	case string:
		return e.EncodeString(v)
	case []byte:
		return e.EncodeBytes(v)
	case int:
		return e.EncodeInt(int64(v))
	case int64:
		return e.EncodeInt(v)
	case uint:
		return e.EncodeUint(uint64(v))
	case uint64:
		return e.EncodeUint(v)
	case bool:
		return e.EncodeBool(v)
	case float32:
		return e.EncodeFloat32(v)
	case float64:
		return e.EncodeFloat64(v)
	case time.Duration:
		return e.EncodeInt(int64(v))
	case time.Time:
		return e.EncodeTime(v)
	}
	return e.EncodeValue(reflect.ValueOf(v))
}

func (e *Encoder) EncodeValue(v reflect.Value) error {
	encode := getEncoder(v.Type())
	return encode(e, v)
}

func (e *Encoder) EncodeNil() error {
	return e.writeCode(codes.Nil)
}

func (e *Encoder) EncodeBool(value bool) error {
	if value {
		return e.writeCode(codes.True)
	}
	return e.writeCode(codes.False)
}

func (e *Encoder) writeCode(c codes.Code) error {
	return e.w.WriteByte(byte(c))
}

func (e *Encoder) write(b []byte) error {
	_, err := e.w.Write(b)
	return err
}

func (e *Encoder) writeString(s string) error {
	_, err := e.w.WriteString(s)
	return err
}
