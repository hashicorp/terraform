package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/reflectwalk"
)

// interpRegexp is a regexp that matches interpolations such as ${foo.bar}
var interpRegexp *regexp.Regexp = regexp.MustCompile(
	`(?i)(\$+)\{([\s*-.,\\/\(\)a-z0-9_"]+)\}`)

// interpolationWalker implements interfaces for the reflectwalk package
// (github.com/mitchellh/reflectwalk) that can be used to automatically
// execute a callback for an interpolation.
type interpolationWalker struct {
	F       interpolationWalkerFunc
	Replace bool

	key         []string
	lastValue   reflect.Value
	loc         reflectwalk.Location
	cs          []reflect.Value
	csData      interface{}
	unknownKeys []string
}

// interpolationWalkerFunc is the callback called by interpolationWalk.
// It is called with any interpolation found. It should return a value
// to replace the interpolation with, along with any errors.
//
// If Replace is set to false in interpolationWalker, then the replace
// value can be anything as it will have no effect.
type interpolationWalkerFunc func(Interpolation) (string, error)

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
	}

	return nil
}

func (w *interpolationWalker) Map(m reflect.Value) error {
	w.cs = append(w.cs, m)
	return nil
}

func (w *interpolationWalker) MapElem(m, k, v reflect.Value) error {
	w.csData = k
	w.key = append(w.key, k.String())
	w.lastValue = v
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

		replaceVal, err := w.F(i)
		if err != nil {
			return fmt.Errorf(
				"%s: %s",
				key,
				err)
		}

		if w.Replace {
			// If this is an unknown variable, then we remove it from
			// the configuration.
			if replaceVal == UnknownVariableValue {
				w.removeCurrent()
				return nil
			}

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
	c := w.cs[len(w.cs)-1]
	switch c.Kind() {
	case reflect.Map:
		// Zero value so that we delete the map key
		var val reflect.Value

		// Get the key and delete it
		k := w.csData.(reflect.Value)
		c.SetMapIndex(k, val)
	}

	// Append the key to the unknown keys
	w.unknownKeys = append(w.unknownKeys, strings.Join(w.key, "."))
}
