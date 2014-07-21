package config

import (
	"reflect"
	"regexp"

	"github.com/mitchellh/reflectwalk"
)

// interpRegexp is a regexp that matches interpolations such as ${foo.bar}
var interpRegexp *regexp.Regexp = regexp.MustCompile(
	`(?i)(\$+)\{([*-.a-z0-9_]+)\}`)

// interpolationWalker implements interfaces for the reflectwalk package
// (github.com/mitchellh/reflectwalk) that can be used to automatically
// execute a callback for an interpolation.
type interpolationWalker struct {
	// F must be one of interpolationWalkerFunc or
	// interpolationReplaceWalkerFunc.
	F       interpolationWalkerFunc
	Replace bool

	key    []string
	loc    reflectwalk.Location
	cs     []reflect.Value
	csData interface{}
}

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
	return nil
}

func (w *interpolationWalker) Primitive(v reflect.Value) error {
	// We only care about strings
	if v.Kind() == reflect.Interface {
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

	for _, match := range matches {
		dollars := len(match[1])

		// If there are even amounts of dollar signs, then it is escaped
		if dollars%2 == 0 {
			continue
		}

		// Interpolation found, instantiate it
		key := match[2]

		i, err := NewInterpolation(key)
		if err != nil {
			return err
		}

		replaceVal, err := w.F(i)
		if err != nil {
			return err
		}

		if w.Replace {
			// TODO(mitchellh): replace
			println(replaceVal)
		}

		return nil
	}

	return nil
}
