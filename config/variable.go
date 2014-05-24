package config

import (
	"reflect"
	"regexp"
	"strings"
)

// varRegexp is a regexp that matches variables such as ${foo.bar}
var varRegexp *regexp.Regexp

func init() {
	varRegexp = regexp.MustCompile(`(?i)(\$+)\{([-.a-z0-9_]+)\}`)
}

// variableDetectWalker implements interfaces for the reflectwalk package
// (github.com/mitchellh/reflectwalk) that can be used to automatically
// pull out the variables that need replacing.
type variableDetectWalker struct {
	Variables map[string]InterpolatedVariable
}

func (w *variableDetectWalker) Primitive(v reflect.Value) error {
	// We only care about strings
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
		if strings.HasPrefix(key, "var.") {
			iv, err = NewUserVariable(key)
		} else {
			iv, err = NewResourceVariable(key)
		}

		if err != nil {
			return err
		}

		w.Variables[key] = iv
	}

	return nil
}
