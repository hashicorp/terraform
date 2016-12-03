package glob

import (
	"regexp"
	"strings"
)

// Glob holds a Unix-style glob pattern in a compiled form for efficient
// matching against paths.
//
// Glob notation:
//  - `?` matches a single char in a single path component
//  - `*` matches zero or more chars in a single path component
//  - `**` matches zero or more chars in zero or more components
//  - any other sequence matches itself
type Glob struct {
	pattern string         // original glob pattern
	regexp  *regexp.Regexp // compiled regexp
}

const charPat = `[^/]`

func mustBuildRe(p string) *regexp.Regexp {
	return regexp.MustCompile(`^/$|^(` + p + `+)?(/` + p + `+)*$`)
}

var globRe = mustBuildRe(`(` + charPat + `|[\*\?])`)

// Supports unix/ruby-style glob patterns:
//  - `?` matches a single char in a single path component
//  - `*` matches zero or more chars in a single path component
//  - `**` matches zero or more chars in zero or more components
func translateGlob(pat string) (string, error) {
	if !globRe.MatchString(pat) {
		return "", Error(pat)
	}

	outs := make([]string, len(pat))
	i, double := 0, false
	for _, c := range pat {
		switch c {
		default:
			outs[i] = string(c)
			double = false
		case '.', '+', '-', '^', '$', '[', ']', '(', ')':
			outs[i] = `\` + string(c)
			double = false
		case '?':
			outs[i] = `[^/]`
			double = false
		case '*':
			if double {
				outs[i-1] = `.*`
			} else {
				outs[i] = `[^/]*`
			}
			double = !double
		}
		i++
	}
	outs = outs[0:i]

	return "^" + strings.Join(outs, "") + "$", nil
}

// CompileGlob translates pat into a form more convenient for
// matching against paths in the store.
func CompileGlob(pat string) (glob Glob, err error) {
	pat = toSlash(pat)
	s, err := translateGlob(pat)
	if err != nil {
		return
	}
	r, err := regexp.Compile(s)
	if err != nil {
		return
	}
	glob = Glob{pat, r}
	return
}

// MustCompileGlob is like CompileGlob, but it panics if an error occurs,
// simplifying safe initialization of global variables holding glob patterns.
func MustCompileGlob(pat string) Glob {
	g, err := CompileGlob(pat)
	if err != nil {
		panic(err)
	}
	return g
}

func (g Glob) String() string {
	return g.pattern
}

func (g Glob) Match(path string) bool {
	return g.regexp.MatchString(toSlash(path))
}

type Error string

func (e Error) Error() string {
	return "invalid glob pattern: " + string(e)
}

func toSlash(path string) string {
	return strings.Replace(path, "\\", "/", -1)
}
