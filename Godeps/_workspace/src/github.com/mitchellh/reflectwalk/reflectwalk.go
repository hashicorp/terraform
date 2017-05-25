// reflectwalk is a package that allows you to "walk" complex structures
// similar to how you may "walk" a filesystem: visiting every element one
// by one and calling callback functions allowing you to handle and manipulate
// those elements.
package reflectwalk

import (
	"reflect"
)

type Location uint

const (
	None Location = iota
	Map
	MapKey
	MapValue
	Slice
	SliceElem
	StructField
	WalkLoc
)

// PrimitiveWalker implementations are able to handle primitive values
// within complex structures. Primitive values are numbers, strings,
// booleans, funcs, chans.
//
// These primitive values are often members of more complex
// structures (slices, maps, etc.) that are walkable by other interfaces.
type PrimitiveWalker interface {
	Primitive(reflect.Value) error
}

// MapWalker implementations are able to handle individual elements
// found within a map structure.
type MapWalker interface {
	Map(m reflect.Value) error
	MapElem(m, k, v reflect.Value) error
}

// SliceWalker implementations are able to handle slice elements found
// within complex structures.
type SliceWalker interface {
	Slice(reflect.Value) error
	SliceElem(int, reflect.Value) error
}

// StructWalker is an interface that has methods that are called for
// structs when a Walk is done.
type StructWalker interface {
	StructField(reflect.StructField, reflect.Value) error
}

// EnterExitWalker implementations are notified before and after
// they walk deeper into complex structures (into struct fields,
// into slice elements, etc.)
type EnterExitWalker interface {
	Enter(Location) error
	Exit(Location) error
}

// Walk takes an arbitrary value and an interface and traverses the
// value, calling callbacks on the interface if they are supported.
// The interface should implement one or more of the walker interfaces
// in this package, such as PrimitiveWalker, StructWalker, etc.
func Walk(data, walker interface{}) (err error) {
	v := reflect.Indirect(reflect.ValueOf(data))
	ew, ok := walker.(EnterExitWalker)
	if ok {
		err = ew.Enter(WalkLoc)
	}

	if err == nil {
		err = walk(v, walker)
	}

	if ok && err == nil {
		err = ew.Exit(WalkLoc)
	}

	return
}

func walk(v reflect.Value, w interface{}) error {
	// We preserve the original value here because if it is an interface
	// type, we want to pass that directly into the walkPrimitive, so that
	// we can set it.
	originalV := v
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	k := v.Kind()
	if k >= reflect.Int && k <= reflect.Complex128 {
		k = reflect.Int
	}

	switch k {
	// Primitives
	case reflect.Bool:
		fallthrough
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.String:
		return walkPrimitive(originalV, w)
	case reflect.Map:
		return walkMap(v, w)
	case reflect.Slice:
		return walkSlice(v, w)
	case reflect.Struct:
		return walkStruct(v, w)
	default:
		panic("unsupported type: " + k.String())
	}
}

func walkMap(v reflect.Value, w interface{}) error {
	ew, ewok := w.(EnterExitWalker)
	if ewok {
		ew.Enter(Map)
	}

	if mw, ok := w.(MapWalker); ok {
		if err := mw.Map(v); err != nil {
			return err
		}
	}

	for _, k := range v.MapKeys() {
		kv := v.MapIndex(k)

		if mw, ok := w.(MapWalker); ok {
			if err := mw.MapElem(v, k, kv); err != nil {
				return err
			}
		}

		ew, ok := w.(EnterExitWalker)
		if ok {
			ew.Enter(MapKey)
		}

		if err := walk(k, w); err != nil {
			return err
		}

		if ok {
			ew.Exit(MapKey)
			ew.Enter(MapValue)
		}

		if err := walk(kv, w); err != nil {
			return err
		}

		if ok {
			ew.Exit(MapValue)
		}
	}

	if ewok {
		ew.Exit(Map)
	}

	return nil
}

func walkPrimitive(v reflect.Value, w interface{}) error {
	if pw, ok := w.(PrimitiveWalker); ok {
		return pw.Primitive(v)
	}

	return nil
}

func walkSlice(v reflect.Value, w interface{}) (err error) {
	ew, ok := w.(EnterExitWalker)
	if ok {
		ew.Enter(Slice)
	}

	if sw, ok := w.(SliceWalker); ok {
		if err := sw.Slice(v); err != nil {
			return err
		}
	}

	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)

		if sw, ok := w.(SliceWalker); ok {
			if err := sw.SliceElem(i, elem); err != nil {
				return err
			}
		}

		ew, ok := w.(EnterExitWalker)
		if ok {
			ew.Enter(SliceElem)
		}

		if err := walk(elem, w); err != nil {
			return err
		}

		if ok {
			ew.Exit(SliceElem)
		}
	}

	ew, ok = w.(EnterExitWalker)
	if ok {
		ew.Exit(Slice)
	}

	return nil
}

func walkStruct(v reflect.Value, w interface{}) (err error) {
	vt := v.Type()
	for i := 0; i < vt.NumField(); i++ {
		sf := vt.Field(i)
		f := v.FieldByIndex([]int{i})

		if sw, ok := w.(StructWalker); ok {
			err = sw.StructField(sf, f)
			if err != nil {
				return
			}
		}

		ew, ok := w.(EnterExitWalker)
		if ok {
			ew.Enter(StructField)
		}

		err = walk(f, w)
		if err != nil {
			return
		}

		if ok {
			ew.Exit(StructField)
		}
	}

	return nil
}
