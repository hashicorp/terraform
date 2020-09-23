// Package dotenv implements the parsing of the .env format.
//
// There is no formal definition of the format but it has been introduced by
// https://github.com/bkeepers/dotenv which is thus canonical.
package dotenv

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// LINE is the regexp matching a single line
const LINE = `
\A
\s*
(?:|#.*|          # comment line
(?:export\s+)?    # optional export
([\w\.]+)         # key
(?:\s*=\s*|:\s+?) # separator
(                 # optional value begin
  '(?:\'|[^'])*'  #   single quoted value
  |               #   or
  "(?:\"|[^"])*"  #   double quoted value
  |               #   or
  [^#\n]+         #   unquoted value
)?                # value end
\s*
(?:\#.*)?         # optional comment
)
\z
`

var linesRe = regexp.MustCompile("[\\r\\n]+")
var lineRe = regexp.MustCompile(
	regexp.MustCompile("\\s+").ReplaceAllLiteralString(
		regexp.MustCompile("\\s+# .*").ReplaceAllLiteralString(LINE, ""), ""))

// Parse reads a string in the .env format and returns a map of the extracted key=values.
//
// Ported from https://github.com/bkeepers/dotenv/blob/84f33f48107c492c3a99bd41c1059e7b4c1bb67a/lib/dotenv/parser.rb
func Parse(data string) (map[string]string, error) {
	var dotenv = make(map[string]string)

	for _, line := range linesRe.Split(data, -1) {
		if !lineRe.MatchString(line) {
			return nil, fmt.Errorf("invalid line: %s", line)
		}

		match := lineRe.FindStringSubmatch(line)
		// commented or empty line
		if len(match) == 0 {
			continue
		}
		if len(match[1]) == 0 {
			continue
		}

		key := match[1]
		value := match[2]

		err := parseValue(key, value, dotenv)

		if err != nil {
			return nil, fmt.Errorf("unable to parse %s, %s: %s", key, value, err)
		}
	}

	return dotenv, nil
}

// MustParse works the same as Parse but panics on error
func MustParse(data string) map[string]string {
	env, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return env
}

func parseValue(key string, value string, dotenv map[string]string) error {
	if len(value) <= 1 {
		dotenv[key] = value
		return nil
	}

	singleQuoted := false

	if value[0:1] == "'" && value[len(value)-1:] == "'" {
		// single-quoted string, do not expand
		singleQuoted = true
		value = value[1 : len(value)-1]
	} else if value[0:1] == `"` && value[len(value)-1:] == `"` {
		value = value[1 : len(value)-1]
		value = expandNewLines(value)
		value = unescapeCharacters(value)
	}

	if !singleQuoted {
		value = expandEnv(value, dotenv)
	}

	dotenv[key] = value
	return nil
}

var escRe = regexp.MustCompile("\\\\([^$])")

func unescapeCharacters(value string) string {
	return escRe.ReplaceAllString(value, "$1")
}

func expandNewLines(value string) string {
	value = strings.Replace(value, "\\n", "\n", -1)
	value = strings.Replace(value, "\\r", "\r", -1)
	return value
}

func expandEnv(value string, dotenv map[string]string) string {
	expander := func(value string) string {
		expanded, found := lookupDotenv(value, dotenv)

		if found {
			return expanded
		} else {
			return os.Getenv(value)
		}
	}

	return os.Expand(value, expander)
}

func lookupDotenv(value string, dotenv map[string]string) (string, bool) {
	retval, ok := dotenv[value]
	return retval, ok
}
