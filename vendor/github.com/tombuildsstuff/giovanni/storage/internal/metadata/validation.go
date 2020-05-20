package metadata

import (
	"fmt"
	"regexp"
	"strings"
)

var cSharpKeywords = map[string]*struct{}{
	"abstract":   {},
	"as":         {},
	"base":       {},
	"bool":       {},
	"break":      {},
	"byte":       {},
	"case":       {},
	"catch":      {},
	"char":       {},
	"checked":    {},
	"class":      {},
	"const":      {},
	"continue":   {},
	"decimal":    {},
	"default":    {},
	"delegate":   {},
	"do":         {},
	"double":     {},
	"else":       {},
	"enum":       {},
	"event":      {},
	"explicit":   {},
	"extern":     {},
	"false":      {},
	"finally":    {},
	"fixed":      {},
	"float":      {},
	"for":        {},
	"foreach":    {},
	"goto":       {},
	"if":         {},
	"implicit":   {},
	"in":         {},
	"int":        {},
	"interface":  {},
	"internal":   {},
	"is":         {},
	"lock":       {},
	"long":       {},
	"namespace":  {},
	"new":        {},
	"null":       {},
	"object":     {},
	"operator":   {},
	"out":        {},
	"override":   {},
	"params":     {},
	"private":    {},
	"protected":  {},
	"public":     {},
	"readonly":   {},
	"ref":        {},
	"return":     {},
	"sbyte":      {},
	"sealed":     {},
	"short":      {},
	"sizeof":     {},
	"stackalloc": {},
	"static":     {},
	"string":     {},
	"struct":     {},
	"switch":     {},
	"this":       {},
	"throw":      {},
	"true":       {},
	"try":        {},
	"typeof":     {},
	"uint":       {},
	"ulong":      {},
	"unchecked":  {},
	"unsafe":     {},
	"ushort":     {},
	"using":      {},
	"void":       {},
	"volatile":   {},
	"while":      {},
}

func Validate(input map[string]string) error {

	for k := range input {
		isCSharpKeyword := cSharpKeywords[strings.ToLower(k)] != nil
		if isCSharpKeyword {
			return fmt.Errorf("%q is not a valid key (C# keyword)", k)
		}

		// must begin with a letter, underscore
		// the rest: letters, digits and underscores
		r, _ := regexp.Compile(`^([A-Za-z_]{1}[A-Za-z0-9_]{1,})$`)
		if !r.MatchString(k) {
			return fmt.Errorf("MetaData must start with letters or an underscores. Got %q.", k)
		}
	}

	return nil
}
