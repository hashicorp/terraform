package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/reflectwalk"
)

// varRegexp is a regexp that matches variables such as ${foo.bar}
var varRegexp *regexp.Regexp

func init() {
	varRegexp = regexp.MustCompile(`(?i)(\$+)\{([*-.a-z0-9_]+)\}`)
}

// ReplaceVariables takes a configuration and a mapping of variables
// and performs the structure walking necessary to properly replace
// all the variables.
func ReplaceVariables(
	c interface{},
	vs map[string]string) ([]string, error) {
	w := &variableReplaceWalker{Values: vs}
	if err := reflectwalk.Walk(c, w); err != nil {
		return nil, err
	}

	return w.UnknownKeys, nil
}

// variableDetectWalker implements interfaces for the reflectwalk package
// (github.com/mitchellh/reflectwalk) that can be used to automatically
// pull out the variables that need replacing.
type variableDetectWalker struct {
	Variables map[string]InterpolatedVariable
}

func (w *variableDetectWalker) Primitive(v reflect.Value) error {
	// We only care about strings
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return nil
	}

	// XXX: This can be a lot more efficient if we used a real
	// parser. A regexp is a hammer though that will get this working.

	matches := varRegexp.FindAllStringSubmatch(v.String(), -1)
	if len(matches) == 0 {
		return nil
	}

	for _, match := range matches {
		dollars := len(match[1])

		// If there are even amounts of dollar signs, then it is escaped
		if dollars%2 == 0 {
			continue
		}

		// Otherwise, record it
		key := match[2]
		if w.Variables == nil {
			w.Variables = make(map[string]InterpolatedVariable)
		}
		if _, ok := w.Variables[key]; ok {
			continue
		}

		var err error
		var iv InterpolatedVariable
		if strings.Index(key, ".") == -1 {
			return fmt.Errorf(
				"Interpolated variable '%s' has bad format. "+
					"Did you mean 'var.%s'?",
				key, key)
		}

		if !strings.HasPrefix(key, "var.") {
			iv, err = NewResourceVariable(key)
		} else {
			varKey := key[len("var."):]
			if strings.Index(varKey, ".") == -1 {
				iv, err = NewUserVariable(key)
			} else {
				iv, err = NewUserMapVariable(key)
			}
		}

		if err != nil {
			return err
		}

		w.Variables[key] = iv
	}

	return nil
}

// variableReplaceWalker implements interfaces for reflectwalk that
// is used to replace variables with their values.
//
// If Values does not have every available value, then the program
// will _panic_. The variableDetectWalker will tell you all variables
// you need.
type variableReplaceWalker struct {
	Variables   map[string]InterpolatedVariable
	Values      map[string]string
	UnknownKeys []string

	key    []string
	loc    reflectwalk.Location
	cs     []reflect.Value
	csData interface{}
}

func (w *variableReplaceWalker) Enter(loc reflectwalk.Location) error {
	w.loc = loc
	return nil
}

func (w *variableReplaceWalker) Exit(loc reflectwalk.Location) error {
	w.loc = reflectwalk.None

	switch loc {
	case reflectwalk.Map:
		w.cs = w.cs[:len(w.cs)-1]
	case reflectwalk.MapValue:
		w.key = w.key[:len(w.key)-1]
	}

	return nil
}

func (w *variableReplaceWalker) Map(m reflect.Value) error {
	w.cs = append(w.cs, m)
	return nil
}

func (w *variableReplaceWalker) MapElem(m, k, v reflect.Value) error {
	w.csData = k
	w.key = append(w.key, k.String())
	return nil
}

func (w *variableReplaceWalker) Primitive(v reflect.Value) error {
	// We only care about strings
	setV := v
	if v.Kind() == reflect.Interface {
		setV = v
		v = v.Elem()
	}
	if v.Kind() != reflect.String {
		return nil
	}

	matches := varRegexp.FindAllStringSubmatch(v.String(), -1)
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

		// Get the key
		key := match[2]
		value, ok := w.Values[key]
		if !ok {
			panic("no value for variable key: " + key)
		}

		// If this is an unknown variable, then we remove it from
		// the configuration.
		if value == UnknownVariableValue {
			w.removeCurrent()
			return nil
		}

		// Replace
		result = strings.Replace(result, match[0], value, -1)
	}

	resultVal := reflect.ValueOf(result)
	if w.loc == reflectwalk.MapValue {
		// If we're in a map, then the only way to set a map value is
		// to set it directly.
		m := w.cs[len(w.cs)-1]
		mk := w.csData.(reflect.Value)
		m.SetMapIndex(mk, resultVal)
	} else {
		// Otherwise, we should be addressable
		setV.Set(resultVal)
	}

	return nil
}

func (w *variableReplaceWalker) removeCurrent() {
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
	w.UnknownKeys = append(w.UnknownKeys, strings.Join(w.key, "."))
}
