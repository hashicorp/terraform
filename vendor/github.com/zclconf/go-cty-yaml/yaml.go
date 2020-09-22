// Package yaml can marshal and unmarshal cty values in YAML format.
package yaml

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/zclconf/go-cty/cty"
)

// Unmarshal reads the document found within the given source buffer
// and attempts to convert it into a value conforming to the given type
// constraint.
//
// This is an alias for Unmarshal on the predefined Converter in "Standard".
//
// An error is returned if the given source contains any YAML document
// delimiters.
func Unmarshal(src []byte, ty cty.Type) (cty.Value, error) {
	return Standard.Unmarshal(src, ty)
}

// Marshal serializes the given value into a YAML document, using a fixed
// mapping from cty types to YAML constructs.
//
// This is an alias for Marshal on the predefined Converter in "Standard".
//
// Note that unlike the function of the same name in the cty JSON package,
// this does not take a type constraint and therefore the YAML serialization
// cannot preserve late-bound type information in the serialization to be
// recovered from Unmarshal. Instead, any cty.DynamicPseudoType in the type
// constraint given to Unmarshal will be decoded as if the corresponding portion
// of the input were processed with ImpliedType to find a target type.
func Marshal(v cty.Value) ([]byte, error) {
	return Standard.Marshal(v)
}

// ImpliedType analyzes the given source code and returns a suitable type that
// it could be decoded into.
//
// For a converter that is using standard YAML rather than cty-specific custom
// tags, only a subset of cty types can be produced: strings, numbers, bools,
// tuple types, and object types.
//
// This is an alias for ImpliedType on the predefined Converter in "Standard".
func ImpliedType(src []byte) (cty.Type, error) {
	return Standard.ImpliedType(src)
}

func handleErr(err *error) {
	if v := recover(); v != nil {
		if e, ok := v.(yamlError); ok {
			*err = e.err
		} else {
			panic(v)
		}
	}
}

type yamlError struct {
	err error
}

func fail(err error) {
	panic(yamlError{err})
}

func failf(format string, args ...interface{}) {
	panic(yamlError{fmt.Errorf("yaml: "+format, args...)})
}

// --------------------------------------------------------------------------
// Maintain a mapping of keys to structure field indexes

// The code in this section was copied from mgo/bson.

// structInfo holds details for the serialization of fields of
// a given struct.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo

	// InlineMap is the number of the field in the struct that
	// contains an ,inline map, or -1 if there's none.
	InlineMap int
}

type fieldInfo struct {
	Key       string
	Num       int
	OmitEmpty bool
	Flow      bool
	// Id holds the unique field identifier, so we can cheaply
	// check for field duplicates without maintaining an extra map.
	Id int

	// Inline holds the field index if the field is part of an inlined struct.
	Inline []int
}

var structMap = make(map[reflect.Type]*structInfo)
var fieldMapMutex sync.RWMutex

func getStructInfo(st reflect.Type) (*structInfo, error) {
	fieldMapMutex.RLock()
	sinfo, found := structMap[st]
	fieldMapMutex.RUnlock()
	if found {
		return sinfo, nil
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	inlineMap := -1
	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("yaml")
		if tag == "" && strings.Index(string(field.Tag), ":") < 0 {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}

		inline := false
		fields := strings.Split(tag, ",")
		if len(fields) > 1 {
			for _, flag := range fields[1:] {
				switch flag {
				case "omitempty":
					info.OmitEmpty = true
				case "flow":
					info.Flow = true
				case "inline":
					inline = true
				default:
					return nil, errors.New(fmt.Sprintf("Unsupported flag %q in tag %q of type %s", flag, tag, st))
				}
			}
			tag = fields[0]
		}

		if inline {
			switch field.Type.Kind() {
			case reflect.Map:
				if inlineMap >= 0 {
					return nil, errors.New("Multiple ,inline maps in struct " + st.String())
				}
				if field.Type.Key() != reflect.TypeOf("") {
					return nil, errors.New("Option ,inline needs a map with string keys in struct " + st.String())
				}
				inlineMap = info.Num
			case reflect.Struct:
				sinfo, err := getStructInfo(field.Type)
				if err != nil {
					return nil, err
				}
				for _, finfo := range sinfo.FieldsList {
					if _, found := fieldsMap[finfo.Key]; found {
						msg := "Duplicated key '" + finfo.Key + "' in struct " + st.String()
						return nil, errors.New(msg)
					}
					if finfo.Inline == nil {
						finfo.Inline = []int{i, finfo.Num}
					} else {
						finfo.Inline = append([]int{i}, finfo.Inline...)
					}
					finfo.Id = len(fieldsList)
					fieldsMap[finfo.Key] = finfo
					fieldsList = append(fieldsList, finfo)
				}
			default:
				//return nil, errors.New("Option ,inline needs a struct value or map field")
				return nil, errors.New("Option ,inline needs a struct value field")
			}
			continue
		}

		if tag != "" {
			info.Key = tag
		} else {
			info.Key = strings.ToLower(field.Name)
		}

		if _, found = fieldsMap[info.Key]; found {
			msg := "Duplicated key '" + info.Key + "' in struct " + st.String()
			return nil, errors.New(msg)
		}

		info.Id = len(fieldsList)
		fieldsList = append(fieldsList, info)
		fieldsMap[info.Key] = info
	}

	sinfo = &structInfo{
		FieldsMap:  fieldsMap,
		FieldsList: fieldsList,
		InlineMap:  inlineMap,
	}

	fieldMapMutex.Lock()
	structMap[st] = sinfo
	fieldMapMutex.Unlock()
	return sinfo, nil
}
