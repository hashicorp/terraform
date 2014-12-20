package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/reflectwalk"
)

// InterpSplitDelim is the delimeter that is looked for to split when
// it is returned. This is a comma right now but should eventually become
// a value that a user is very unlikely to use (such as UUID).
const InterpSplitDelim = `B780FFEC-B661-4EB8-9236-A01737AD98B6`

// interpRegexp is a regexp that matches interpolations such as ${foo.bar}
var interpRegexp *regexp.Regexp = regexp.MustCompile(
	`(?i)(\$+)\{([\s*-.,\\/\(\):a-z0-9_"]+)\}`)

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
type interpolationWalkerFunc func(Interpolation) (string, error)

// interpolationWalkerContextFunc is called by interpolationWalk if
// ContextF is set. This receives both the interpolation and the location
// where the interpolation is.
//
// This callback can be used to validate the location of the interpolation
// within the configuration.
type interpolationWalkerContextFunc func(reflectwalk.Location, Interpolation)

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

	// XXX: This can be a lot more efficient if we used a real
	// parser. A regexp is a hammer though that will get this working.

	matches := interpRegexp.FindAllStringSubmatch(v.String(), -1)
	if len(matches) == 0 {
		return nil
	}

	result := v.String()
	for _, match := range matches {
		dollars := len(match[1])

		// If there are even amounts of dollar signs, then it is escaped
		if dollars%2 == 0 {
			continue
		}

		// Interpolation found, instantiate it
		key := match[2]

		i, err := ExprParse(key)
		if err != nil {
			return err
		}

		if w.ContextF != nil {
			w.ContextF(w.loc, i)
		}

		if w.F == nil {
			continue
		}

		replaceVal, err := w.F(i)
		if err != nil {
			return fmt.Errorf(
				"%s: %s",
				key,
				err)
		}

		if w.Replace {
			// We need to determine if we need to remove this element
			// if the result contains any "UnknownVariableValue" which is
			// set if it is computed. This behavior is different if we're
			// splitting (in a SliceElem) or not.
			remove := false
			if w.loc == reflectwalk.SliceElem {
				parts := strings.Split(replaceVal, InterpSplitDelim)
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

			// Replace in our interpolation and continue on.
			result = strings.Replace(result, match[0], replaceVal, -1)
		}
	}

	if w.Replace {
		resultVal := reflect.ValueOf(result)
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
		if idx := strings.Index(sv, InterpSplitDelim); idx >= 0 {
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

		// Split on the delimiter
		for _, p := range strings.Split(sv, InterpSplitDelim) {
			result = append(result, p)
		}
	}

	// Our slice is now done, we have to replace the slice now
	// with this new one that we have.
	w.replaceCurrent(reflect.ValueOf(result))
}
