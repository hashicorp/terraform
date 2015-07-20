package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform/config/lang"
	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/mitchellh/reflectwalk"
)

// interpolationWalker implements interfaces for the reflectwalk package
// (github.com/mitchellh/reflectwalk) that can be used to automatically
// execute a callback for an interpolation.
type interpolationWalker struct {
	// F is the function to call for every interpolation. It can be nil.
	//
	// If Replace is true, then the return value of F will be used to
	// replace the interpolation.
	F       interpolationWalkerFunc
	Replace bool

	// ContextF is an advanced version of F that also receives the
	// location of where it is in the structure. This lets you do
	// context-aware validation.
	ContextF interpolationWalkerContextFunc

	key         []string
	lastValue   reflect.Value
	loc         reflectwalk.Location
	cs          []reflect.Value
	csKey       []reflect.Value
	csData      interface{}
	sliceIndex  int
	unknownKeys []string
}

// interpolationWalkerFunc is the callback called by interpolationWalk.
// It is called with any interpolation found. It should return a value
// to replace the interpolation with, along with any errors.
//
// If Replace is set to false in interpolationWalker, then the replace
// value can be anything as it will have no effect.
type interpolationWalkerFunc func(ast.Node) (string, error)

// interpolationWalkerContextFunc is called by interpolationWalk if
// ContextF is set. This receives both the interpolation and the location
// where the interpolation is.
//
// This callback can be used to validate the location of the interpolation
// within the configuration.
type interpolationWalkerContextFunc func(reflectwalk.Location, ast.Node)

func (w *interpolationWalker) Enter(loc reflectwalk.Location) error {
	w.loc = loc
	return nil
}

func (w *interpolationWalker) Exit(loc reflectwalk.Location) error {
	w.loc = reflectwalk.None

	switch loc {
	case reflectwalk.Map:
		w.cs = w.cs[:len(w.cs)-1]
	case reflectwalk.MapValue:
		w.key = w.key[:len(w.key)-1]
		w.csKey = w.csKey[:len(w.csKey)-1]
	case reflectwalk.Slice:
		// Split any values that need to be split
		w.splitSlice()
		w.cs = w.cs[:len(w.cs)-1]
	case reflectwalk.SliceElem:
		w.csKey = w.csKey[:len(w.csKey)-1]
	}

	return nil
}

func (w *interpolationWalker) Map(m reflect.Value) error {
	w.cs = append(w.cs, m)
	return nil
}

func (w *interpolationWalker) MapElem(m, k, v reflect.Value) error {
	w.csData = k
	w.csKey = append(w.csKey, k)
	w.key = append(w.key, k.String())
	w.lastValue = v
	return nil
}

func (w *interpolationWalker) Slice(s reflect.Value) error {
	w.cs = append(w.cs, s)
	return nil
}

func (w *interpolationWalker) SliceElem(i int, elem reflect.Value) error {
	w.csKey = append(w.csKey, reflect.ValueOf(i))
	w.sliceIndex = i
	return nil
}

func (w *interpolationWalker) Primitive(v reflect.Value) error {
	setV := v

	// We only care about strings
	if v.Kind() == reflect.Interface {
		setV = v
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return nil
	}

	astRoot, err := lang.Parse(v.String())
	if err != nil {
		return err
	}

	// If the AST we got is just a literal string value, then we ignore it
	if _, ok := astRoot.(*ast.LiteralNode); ok {
		return nil
	}

	if w.ContextF != nil {
		w.ContextF(w.loc, astRoot)
	}

	if w.F == nil {
		return nil
	}

	replaceVal, err := w.F(astRoot)
	if err != nil {
		return fmt.Errorf(
			"%s in:\n\n%s",
			err, v.String())
	}

	if w.Replace {
		// We need to determine if we need to remove this element
		// if the result contains any "UnknownVariableValue" which is
		// set if it is computed. This behavior is different if we're
		// splitting (in a SliceElem) or not.
		remove := false
		if w.loc == reflectwalk.SliceElem && IsStringList(replaceVal) {
			parts := StringList(replaceVal).Slice()
			for _, p := range parts {
				if p == UnknownVariableValue {
					remove = true
					break
				}
			}
		} else if replaceVal == UnknownVariableValue {
			remove = true
		}
		if remove {
			w.removeCurrent()
			return nil
		}

		resultVal := reflect.ValueOf(replaceVal)
		switch w.loc {
		case reflectwalk.MapKey:
			m := w.cs[len(w.cs)-1]

			// Delete the old value
			var zero reflect.Value
			m.SetMapIndex(w.csData.(reflect.Value), zero)

			// Set the new key with the existing value
			m.SetMapIndex(resultVal, w.lastValue)

			// Set the key to be the new key
			w.csData = resultVal
		case reflectwalk.MapValue:
			// If we're in a map, then the only way to set a map value is
			// to set it directly.
			m := w.cs[len(w.cs)-1]
			mk := w.csData.(reflect.Value)
			m.SetMapIndex(mk, resultVal)
		default:
			// Otherwise, we should be addressable
			setV.Set(resultVal)
		}
	}

	return nil
}

func (w *interpolationWalker) removeCurrent() {
	// Append the key to the unknown keys
	w.unknownKeys = append(w.unknownKeys, strings.Join(w.key, "."))

	for i := 1; i <= len(w.cs); i++ {
		c := w.cs[len(w.cs)-i]
		switch c.Kind() {
		case reflect.Map:
			// Zero value so that we delete the map key
			var val reflect.Value

			// Get the key and delete it
			k := w.csData.(reflect.Value)
			c.SetMapIndex(k, val)
			return
		}
	}

	panic("No container found for removeCurrent")
}

func (w *interpolationWalker) replaceCurrent(v reflect.Value) {
	c := w.cs[len(w.cs)-2]
	switch c.Kind() {
	case reflect.Map:
		// Get the key and delete it
		k := w.csKey[len(w.csKey)-1]
		c.SetMapIndex(k, v)
	}
}

func (w *interpolationWalker) splitSlice() {
	// Get the []interface{} slice so we can do some operations on
	// it without dealing with reflection. We'll document each step
	// here to be clear.
	var s []interface{}
	raw := w.cs[len(w.cs)-1]
	switch v := raw.Interface().(type) {
	case []interface{}:
		s = v
	case []map[string]interface{}:
		return
	default:
		panic("Unknown kind: " + raw.Kind().String())
	}

	// Check if we have any elements that we need to split. If not, then
	// just return since we're done.
	split := false
	for _, v := range s {
		sv, ok := v.(string)
		if !ok {
			continue
		}
		if IsStringList(sv) {
			split = true
			break
		}
	}
	if !split {
		return
	}

	// Make a new result slice that is twice the capacity to fit our growth.
	result := make([]interface{}, 0, len(s)*2)

	// Go over each element of the original slice and start building up
	// the resulting slice by splitting where we have to.
	for _, v := range s {
		sv, ok := v.(string)
		if !ok {
			// Not a string, so just set it
			result = append(result, v)
			continue
		}

		if IsStringList(sv) {
			for _, p := range StringList(sv).Slice() {
				result = append(result, p)
			}
			continue
		}

		// Not a string list, so just set it
		result = append(result, sv)
	}

	// Our slice is now done, we have to replace the slice now
	// with this new one that we have.
	w.replaceCurrent(reflect.ValueOf(result))
}
