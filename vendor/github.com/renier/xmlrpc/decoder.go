package xmlrpc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

const (
	iso8601         = "20060102T15:04:05"
	iso8601hyphen   = "2006-01-02T15:04:05Z"
	iso8601hyphenTZ = "2006-01-02T15:04:05-07:00"
)

var (
	// CharsetReader is a function to generate reader which converts a non UTF-8
	// charset into UTF-8.
	CharsetReader func(string, io.Reader) (io.Reader, error)

	invalidXmlError = errors.New("invalid xml")

	dateFormats = []string{iso8601, iso8601hyphen, iso8601hyphenTZ}

	topArrayRE = regexp.MustCompile(`^<\?xml version="1.0" encoding=".+"\?>\s*<params>\s*<param>\s*<value>\s*<array>`)
)

type TypeMismatchError string

func (e TypeMismatchError) Error() string { return string(e) }

type decoder struct {
	*xml.Decoder
}

func unmarshal(data []byte, v interface{}) (err error) {
	dec := &decoder{xml.NewDecoder(bytes.NewBuffer(data))}

	if CharsetReader != nil {
		dec.CharsetReader = CharsetReader
	} else {
		dec.CharsetReader = defaultCharsetReader
	}

	var tok xml.Token
	for {
		if tok, err = dec.Token(); err != nil {
			return err
		}

		if t, ok := tok.(xml.StartElement); ok {
			if t.Name.Local == "value" {
				val := reflect.ValueOf(v)
				if val.Kind() != reflect.Ptr {
					return errors.New("non-pointer value passed to unmarshal")
				}

				val = val.Elem()
				// Some APIs that normally return a collection, omit the []'s when
				// the API returns a single value.
				if val.Kind() == reflect.Slice && !topArrayRE.MatchString(string(data)) {
					val.Set(reflect.MakeSlice(val.Type(), 1, 1))
					val = val.Index(0)
				}

				if err = dec.decodeValue(val); err != nil {
					return err
				}

				break
			}
		}
	}

	// read until end of document
	err = dec.Skip()
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (dec *decoder) decodeValue(val reflect.Value) error {
	var tok xml.Token
	var err error

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		val = val.Elem()
	}

	var typeName string
	for {
		if tok, err = dec.Token(); err != nil {
			return err
		}

		if t, ok := tok.(xml.EndElement); ok {
			if t.Name.Local == "value" {
				return nil
			} else {
				return invalidXmlError
			}
		}

		if t, ok := tok.(xml.StartElement); ok {
			typeName = t.Name.Local
			break
		}

		// Treat value data without type identifier as string
		if t, ok := tok.(xml.CharData); ok {
			if value := strings.TrimSpace(string(t)); value != "" {
				if err = checkType(val, reflect.String); err != nil {
					return err
				}

				val.SetString(value)
				return nil
			}
		}
	}

	switch typeName {
	case "struct":
		ismap := false
		pmap := val
		valType := val.Type()

		if err = checkType(val, reflect.Struct); err != nil {
			if checkType(val, reflect.Map) == nil {
				if valType.Key().Kind() != reflect.String {
					return fmt.Errorf("only maps with string key type can be unmarshalled")
				}
				ismap = true
			} else if checkType(val, reflect.Interface) == nil && val.IsNil() {
				var dummy map[string]interface{}
				pmap = reflect.New(reflect.TypeOf(dummy)).Elem()
				valType = pmap.Type()
				ismap = true
			} else {
				return err
			}
		}

		var fields map[string]reflect.Value

		if !ismap {
			fields = make(map[string]reflect.Value)
			buildStructFieldMap(&fields, val)
		} else {
			// Create initial empty map
			pmap.Set(reflect.MakeMap(valType))
		}

		// Process struct members.
	StructLoop:
		for {
			if tok, err = dec.Token(); err != nil {
				return err
			}
			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local != "member" {
					return invalidXmlError
				}

				tagName, fieldName, err := dec.readTag()
				if err != nil {
					return err
				}
				if tagName != "name" {
					return invalidXmlError
				}

				var fv reflect.Value
				ok := true

				if !ismap {
					fv, ok = fields[string(fieldName)]
				} else {
					fv = reflect.New(valType.Elem())
				}

				if ok {
					for {
						if tok, err = dec.Token(); err != nil {
							return err
						}
						if t, ok := tok.(xml.StartElement); ok && t.Name.Local == "value" {
							if err = dec.decodeValue(fv); err != nil {
								return err
							}

							// </value>
							if err = dec.Skip(); err != nil {
								return err
							}

							break
						}
					}
				}

				// </member>
				if err = dec.Skip(); err != nil {
					return err
				}

				if ismap {
					pmap.SetMapIndex(reflect.ValueOf(string(fieldName)), reflect.Indirect(fv))
					val.Set(pmap)
				}
			case xml.EndElement:
				break StructLoop
			}
		}
	case "array":
		pslice := val
		if checkType(val, reflect.Interface) == nil && val.IsNil() {
			var dummy []interface{}
			pslice = reflect.New(reflect.TypeOf(dummy)).Elem()
		} else if err = checkType(val, reflect.Slice); err != nil {
			return err
		}

	ArrayLoop:
		for {
			if tok, err = dec.Token(); err != nil {
				return err
			}

			switch t := tok.(type) {
			case xml.StartElement:
				if t.Name.Local != "data" {
					return invalidXmlError
				}

				slice := reflect.MakeSlice(pslice.Type(), 0, 0)

			DataLoop:
				for {
					if tok, err = dec.Token(); err != nil {
						return err
					}

					switch tt := tok.(type) {
					case xml.StartElement:
						if tt.Name.Local != "value" {
							return invalidXmlError
						}

						v := reflect.New(pslice.Type().Elem())
						if err = dec.decodeValue(v); err != nil {
							return err
						}

						slice = reflect.Append(slice, v.Elem())

						// </value>
						if err = dec.Skip(); err != nil {
							return err
						}
					case xml.EndElement:
						pslice.Set(slice)
						val.Set(pslice)
						break DataLoop
					}
				}
			case xml.EndElement:
				break ArrayLoop
			}
		}
	default:
		if tok, err = dec.Token(); err != nil {
			return err
		}

		var data []byte

		switch t := tok.(type) {
		case xml.EndElement:
			return nil
		case xml.CharData:
			data = []byte(t.Copy())
		default:
			return invalidXmlError
		}

	ParseValue:
		switch typeName {
		case "int", "i4", "i8":
			if checkType(val, reflect.Interface) == nil && val.IsNil() {
				i, err := strconv.ParseInt(string(data), 10, 64)
				if err != nil {
					return err
				}

				pi := reflect.New(reflect.TypeOf(i)).Elem()
				pi.SetInt(i)
				val.Set(pi)
			} else if err = checkType(val, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64); err != nil {
				return err
			} else {
				k := val.Kind()
				isInt := k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64

				if isInt {
					i, err := strconv.ParseInt(string(data), 10, val.Type().Bits())
					if err != nil {
						return err
					}

					val.SetInt(i)
				} else {
					i, err := strconv.ParseUint(string(data), 10, val.Type().Bits())
					if err != nil {
						return err
					}

					val.SetUint(i)
				}
			}
		case "string", "base64":
			str := string(data)
			if checkType(val, reflect.Interface) == nil && val.IsNil() {
				pstr := reflect.New(reflect.TypeOf(str)).Elem()
				pstr.SetString(str)
				val.Set(pstr)
			} else if err = checkType(val, reflect.String); err != nil {
				valName := val.Type().Name()
				if valName == "" {
					valName = reflect.Indirect(val).Type().Name()
				}

				if valName == "Time" {
					timeField := val.FieldByName(valName)
					if timeField.IsValid() {
						val = timeField
					}
					typeName = "dateTime.iso8601"
					goto ParseValue
				} else if strings.HasPrefix(strings.ToLower(valName), "float") {
					typeName = "double"
					goto ParseValue
				}
				return err
			} else {
				val.SetString(str)
			}
		case "dateTime.iso8601":
			err = nil
			var t time.Time
			for _, df := range dateFormats {
				t, err = time.Parse(df, string(data))
				if err == nil {
					break
				}
			}
			if err != nil {
				return err
			}

			if checkType(val, reflect.Interface) == nil && val.IsNil() {
				ptime := reflect.New(reflect.TypeOf(t)).Elem()
				ptime.Set(reflect.ValueOf(t))
				val.Set(ptime)
			} else if !reflect.TypeOf((time.Time)(t)).ConvertibleTo(val.Type()) {
				return TypeMismatchError(
					fmt.Sprintf(
						"error: type mismatch error - can't decode %v (%s.%s) to time",
						val.Kind(),
						val.Type().PkgPath(),
						val.Type().Name(),
					),
				)
			} else {
				val.Set(reflect.ValueOf(t).Convert(val.Type()))
			}
		case "boolean":
			v, err := strconv.ParseBool(string(data))
			if err != nil {
				return err
			}

			if checkType(val, reflect.Interface) == nil && val.IsNil() {
				pv := reflect.New(reflect.TypeOf(v)).Elem()
				pv.SetBool(v)
				val.Set(pv)
			} else if err = checkType(val, reflect.Bool); err != nil {
				return err
			} else {
				val.SetBool(v)
			}
		case "double":
			if checkType(val, reflect.Interface) == nil && val.IsNil() {
				i, err := strconv.ParseFloat(string(data), 64)
				if err != nil {
					return err
				}

				pdouble := reflect.New(reflect.TypeOf(i)).Elem()
				pdouble.SetFloat(i)
				val.Set(pdouble)
			} else if err = checkType(val, reflect.Float32, reflect.Float64); err != nil {
				return err
			} else {
				i, err := strconv.ParseFloat(string(data), val.Type().Bits())
				if err != nil {
					return err
				}

				val.SetFloat(i)
			}
		default:
			return errors.New("unsupported type")
		}

		// </type>
		if err = dec.Skip(); err != nil {
			return err
		}
	}

	return nil
}

func (dec *decoder) readTag() (string, []byte, error) {
	var tok xml.Token
	var err error

	var name string
	for {
		if tok, err = dec.Token(); err != nil {
			return "", nil, err
		}

		if t, ok := tok.(xml.StartElement); ok {
			name = t.Name.Local
			break
		}
	}

	value, err := dec.readCharData()
	if err != nil {
		return "", nil, err
	}

	return name, value, dec.Skip()
}

func (dec *decoder) readCharData() ([]byte, error) {
	var tok xml.Token
	var err error

	if tok, err = dec.Token(); err != nil {
		return nil, err
	}

	if t, ok := tok.(xml.CharData); ok {
		return []byte(t.Copy()), nil
	} else {
		return nil, invalidXmlError
	}
}

func checkType(val reflect.Value, kinds ...reflect.Kind) error {
	if len(kinds) == 0 {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	match := false

	for _, kind := range kinds {
		if val.Kind() == kind {
			match = true
			break
		}
	}

	if !match {
		return TypeMismatchError(fmt.Sprintf("error: type mismatch - can't unmarshal %v to %v",
			val.Kind(), kinds[0]))
	}

	return nil
}

func buildStructFieldMap(fieldMap *map[string]reflect.Value, val reflect.Value) {
	valType := val.Type()
	valFieldNum := valType.NumField()
	for i := 0; i < valFieldNum; i++ {
		field := valType.Field(i)
		fieldVal := val.FieldByName(field.Name)

		if field.Anonymous {
			// Drill down into embedded structs
			buildStructFieldMap(fieldMap, fieldVal)
			continue
		}

		if fieldVal.CanSet() {
			if fn := field.Tag.Get("xmlrpc"); fn != "" {
				fn = strings.Split(fn, ",")[0]
				(*fieldMap)[fn] = fieldVal
			} else {
				(*fieldMap)[field.Name] = fieldVal
			}
		}
	}
}

// http://stackoverflow.com/a/34712322/3160958
// https://groups.google.com/forum/#!topic/golang-nuts/VudK_05B62k
func defaultCharsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "iso-8859-1" || charset == "ISO-8859-1" {
		return charmap.ISO8859_1.NewDecoder().Reader(input), nil
	} else if strings.HasPrefix(charset, "utf") || strings.HasPrefix(charset, "UTF") {
		return input, nil
	}

	return nil, fmt.Errorf("Unknown charset: %s", charset)
}
