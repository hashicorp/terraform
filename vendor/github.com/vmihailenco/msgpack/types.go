package msgpack

import (
	"reflect"
	"sync"
)

var errorType = reflect.TypeOf((*error)(nil)).Elem()

var customEncoderType = reflect.TypeOf((*CustomEncoder)(nil)).Elem()
var customDecoderType = reflect.TypeOf((*CustomDecoder)(nil)).Elem()

var marshalerType = reflect.TypeOf((*Marshaler)(nil)).Elem()
var unmarshalerType = reflect.TypeOf((*Unmarshaler)(nil)).Elem()

type encoderFunc func(*Encoder, reflect.Value) error
type decoderFunc func(*Decoder, reflect.Value) error

var typEncMap = make(map[reflect.Type]encoderFunc)
var typDecMap = make(map[reflect.Type]decoderFunc)

// Register registers encoder and decoder functions for a value.
// This is low level API and in most cases you should prefer implementing
// Marshaler/CustomEncoder and Unmarshaler/CustomDecoder interfaces.
func Register(value interface{}, enc encoderFunc, dec decoderFunc) {
	typ := reflect.TypeOf(value)
	if enc != nil {
		typEncMap[typ] = enc
	}
	if dec != nil {
		typDecMap[typ] = dec
	}
}

//------------------------------------------------------------------------------

var structs = newStructCache(false)
var jsonStructs = newStructCache(true)

type structCache struct {
	mu sync.RWMutex
	m  map[reflect.Type]*fields

	useJSONTag bool
}

func newStructCache(useJSONTag bool) *structCache {
	return &structCache{
		m: make(map[reflect.Type]*fields),

		useJSONTag: useJSONTag,
	}
}

func (m *structCache) Fields(typ reflect.Type) *fields {
	m.mu.RLock()
	fs, ok := m.m[typ]
	m.mu.RUnlock()
	if ok {
		return fs
	}

	m.mu.Lock()
	fs, ok = m.m[typ]
	if !ok {
		fs = getFields(typ, m.useJSONTag)
		m.m[typ] = fs
	}
	m.mu.Unlock()

	return fs
}

//------------------------------------------------------------------------------

type field struct {
	name      string
	index     []int
	omitEmpty bool
	encoder   encoderFunc
	decoder   decoderFunc
}

func (f *field) value(v reflect.Value) reflect.Value {
	return fieldByIndex(v, f.index)
}

func (f *field) Omit(strct reflect.Value) bool {
	return f.omitEmpty && isEmptyValue(f.value(strct))
}

func (f *field) EncodeValue(e *Encoder, strct reflect.Value) error {
	return f.encoder(e, f.value(strct))
}

func (f *field) DecodeValue(d *Decoder, strct reflect.Value) error {
	return f.decoder(d, f.value(strct))
}

//------------------------------------------------------------------------------

type fields struct {
	Table   map[string]*field
	List    []*field
	AsArray bool

	hasOmitEmpty bool
}

func newFields(numField int) *fields {
	return &fields{
		Table: make(map[string]*field, numField),
		List:  make([]*field, 0, numField),
	}
}

func (fs *fields) Add(field *field) {
	fs.Table[field.name] = field
	fs.List = append(fs.List, field)
	if field.omitEmpty {
		fs.hasOmitEmpty = true
	}
}

func (fs *fields) OmitEmpty(strct reflect.Value) []*field {
	if !fs.hasOmitEmpty {
		return fs.List
	}

	fields := make([]*field, 0, len(fs.List))
	for _, f := range fs.List {
		if !f.Omit(strct) {
			fields = append(fields, f)
		}
	}
	return fields
}

func getFields(typ reflect.Type, useJSONTag bool) *fields {
	numField := typ.NumField()
	fs := newFields(numField)

	var omitEmpty bool
	for i := 0; i < numField; i++ {
		f := typ.Field(i)

		tag := f.Tag.Get("msgpack")
		if useJSONTag && tag == "" {
			tag = f.Tag.Get("json")
		}

		name, opt := parseTag(tag)
		if name == "-" {
			continue
		}

		if f.Name == "_msgpack" {
			if opt.Contains("asArray") {
				fs.AsArray = true
			}
			if opt.Contains("omitempty") {
				omitEmpty = true
			}
		}

		if f.PkgPath != "" && !f.Anonymous {
			continue
		}

		field := &field{
			name:      name,
			index:     f.Index,
			omitEmpty: omitEmpty || opt.Contains("omitempty"),
			encoder:   getEncoder(f.Type),
			decoder:   getDecoder(f.Type),
		}

		if field.name == "" {
			field.name = f.Name
		}

		if f.Anonymous && !opt.Contains("noinline") {
			inline := opt.Contains("inline")
			if inline {
				inlineFields(fs, f.Type, field, useJSONTag)
			} else {
				inline = autoinlineFields(fs, f.Type, field, useJSONTag)
			}
			if inline {
				fs.Table[field.name] = field
				continue
			}
		}

		fs.Add(field)
	}
	return fs
}

var encodeStructValuePtr uintptr
var decodeStructValuePtr uintptr

func init() {
	encodeStructValuePtr = reflect.ValueOf(encodeStructValue).Pointer()
	decodeStructValuePtr = reflect.ValueOf(decodeStructValue).Pointer()
}

func inlineFields(fs *fields, typ reflect.Type, f *field, useJSONTag bool) {
	inlinedFields := getFields(typ, useJSONTag).List
	for _, field := range inlinedFields {
		if _, ok := fs.Table[field.name]; ok {
			// Don't inline shadowed fields.
			continue
		}
		field.index = append(f.index, field.index...)
		fs.Add(field)
	}
}

func autoinlineFields(fs *fields, typ reflect.Type, f *field, useJSONTag bool) bool {
	var encoder encoderFunc
	var decoder decoderFunc

	if typ.Kind() == reflect.Struct {
		encoder = f.encoder
		decoder = f.decoder
	} else {
		for typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
			encoder = getEncoder(typ)
			decoder = getDecoder(typ)
		}
		if typ.Kind() != reflect.Struct {
			return false
		}
	}

	if reflect.ValueOf(encoder).Pointer() != encodeStructValuePtr {
		return false
	}
	if reflect.ValueOf(decoder).Pointer() != decodeStructValuePtr {
		return false
	}

	inlinedFields := getFields(typ, useJSONTag).List
	for _, field := range inlinedFields {
		if _, ok := fs.Table[field.name]; ok {
			// Don't auto inline if there are shadowed fields.
			return false
		}
	}

	for _, field := range inlinedFields {
		field.index = append(f.index, field.index...)
		fs.Add(field)
	}
	return true
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	if len(index) == 1 {
		return v.Field(index[0])
	}
	for i, x := range index {
		if i > 0 {
			var ok bool
			v, ok = indirectNew(v)
			if !ok {
				return v
			}
		}
		v = v.Field(x)
	}
	return v
}

func indirectNew(v reflect.Value) (reflect.Value, bool) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			if !v.CanSet() {
				return v, false
			}
			elemType := v.Type().Elem()
			if elemType.Kind() != reflect.Struct {
				return v, false
			}
			v.Set(reflect.New(elemType))
		}
		v = v.Elem()
	}
	return v, true
}
