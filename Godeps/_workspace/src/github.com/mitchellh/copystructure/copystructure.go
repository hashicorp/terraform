package copystructure

import (
	"reflect"

	"github.com/mitchellh/reflectwalk"
)

// Copy returns a deep copy of v.
func Copy(v interface{}) (interface{}, error) {
	w := new(walker)
	err := reflectwalk.Walk(v, w)
	if err != nil {
		return nil, err
	}

	return w.Result, nil
}

type walker struct {
	Result interface{}

	vals []reflect.Value
	cs   []reflect.Value
}

func (w *walker) Enter(l reflectwalk.Location) error {
	return nil
}

func (w *walker) Exit(l reflectwalk.Location) error {
	switch l {
	case reflectwalk.Map:
		fallthrough
	case reflectwalk.Slice:
		// Pop map off our container
		w.cs = w.cs[:len(w.cs)-1]
	case reflectwalk.MapValue:
		// Pop off the key and value
		mv := w.valPop()
		mk := w.valPop()
		m := w.cs[len(w.cs)-1]
		m.SetMapIndex(mk, mv)
	case reflectwalk.SliceElem:
		// Pop off the value and the index and set it on the slice
		v := w.valPop()
		i := w.valPop().Interface().(int)
		s := w.cs[len(w.cs)-1]
		s.Index(i).Set(v)

	case reflectwalk.WalkLoc:
		// Clear out the slices for GC
		w.cs = nil
		w.vals = nil
	}

	return nil
}

func (w *walker) Map(m reflect.Value) error {
	t := m.Type()
	newMap := reflect.MakeMap(reflect.MapOf(t.Key(), t.Elem()))
	w.cs = append(w.cs, newMap)
	w.valPush(newMap)
	return nil
}

func (w *walker) MapElem(m, k, v reflect.Value) error {
	return nil
}

func (w *walker) Primitive(v reflect.Value) error {
	w.valPush(v)
	return nil
}

func (w *walker) Slice(s reflect.Value) error {
	newS := reflect.MakeSlice(s.Type(), s.Len(), s.Cap())
	w.cs = append(w.cs, newS)
	w.valPush(newS)
	return nil
}

func (w *walker) SliceElem(i int, elem reflect.Value) error {
	// We don't write the slice here because elem might still be
	// arbitrarily complex. Just record the index and continue on.
	w.valPush(reflect.ValueOf(i))

	return nil
}

func (w *walker) valPop() reflect.Value {
	result := w.vals[len(w.vals)-1]
	w.vals = w.vals[:len(w.vals)-1]
	return result
}

func (w *walker) valPush(v reflect.Value) {
	w.vals = append(w.vals, v)

	// If we haven't set the result yet, then this is the result since
	// it is the first (outermost) value we're seeing.
	if w.Result == nil {
		w.Result = v.Interface()
	}
}
