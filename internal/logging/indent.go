package logging

import (
	"strings"
)

// Indent adds two spaces to the beginning of each line of the given string,
// with the goal of making the log level filter understand it as a line
// continuation rather than possibly as new log lines.
func Indent(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		end := strings.IndexByte(s, '\n')
		if end == -1 {
			end = len(s) - 1
		}
		var l string
		l, s = s[:end+1], s[end+1:]
		b.WriteString("  ")
		b.WriteString(l)
	}
	return b.String()
}
