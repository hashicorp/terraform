package shquot

import (
	"strings"
)

// These lists of characters are taken from IEEE Std 1003.1, 2004 Edition
// http://pubs.opengroup.org/onlinepubs/009696899/utilities/xcu_chap02.html
//
// The distinction between "meta" and "space" here is not part of the standard
// and is instead how we decide whether to use backslash escapes or single
// quotes for a particular word. The newline character is an exception, however:
// it cannot be backslash-escaped in the usual way, so it must _always_ trigger
// our single-quote behavior for correct results.
const (
	posixMeta  = "|&;<>(){}[]$`\\\"'*?!#~=%"
	posixSpace = " \n\t"
)

// POSIXShell quotes the given command line for interpretation by shells
// compatible with the POSIX shell standards, including most superset
// implementations like bash.
//
// It will pass through individual arguments unchanged where possible for
// maximum readability. It will use backslash escapes for arguments that do
// not contain whitespace, and single quotes for arguments that do.
//
// This function assumes a shell with the default value of the "IFS" variable,
// such that a single space will be interpreted as an argument separator.
//
// The first argument is always quoted so that it will bypass alias expansion
// and function call behaviors in compliant shells.
func POSIXShell(cmdline []string) string {
	if len(cmdline) == 0 {
		return ""
	}

	var buf strings.Builder
	posixShellSingleQuoted(cmdline[0], &buf)
	for _, a := range cmdline[1:] {
		buf.WriteByte(' ')
		posixShellSingle(a, &buf)
	}
	return buf.String()
}

// POSIXShellSplit is a variant of POSIXShell that isolates the first argument
// (conventionally the program name) and returns it verbatim along with a quoted
// version of the remaining arguments.
func POSIXShellSplit(cmdline []string) (cmd, args string) {
	if len(cmdline) == 0 {
		return "", ""
	}

	var buf strings.Builder
	for i, a := range cmdline[1:] {
		if i > 0 {
			buf.WriteByte(' ')
		}
		posixShellSingle(a, &buf)
	}
	return cmdline[0], buf.String()
}

func posixShellSingle(a string, to *strings.Builder) {
	if len(a) == 0 {
		to.WriteString("''")
		return
	}

	switch {
	case strings.ContainsAny(a, posixSpace):
		posixShellSingleQuoted(a, to)
	default:
		posixShellSingleBackslash(a, to)
	}
}

func posixShellSingleBackslash(a string, to *strings.Builder) {
	litLen := 0
	remain := a
	for len(remain) > 0 {
		var this rune
		this, remain = rune(remain[0]), remain[1:]

		if strings.ContainsRune(posixMeta, this) {
			// If we've skipped over any literal characters then we need to
			// emit them first.
			if litLen > 0 {
				to.WriteString(a[:litLen])
				litLen = 0
			}
			to.WriteByte('\\')
			to.WriteRune(this)
			a = remain // resync "a" to avoid treating this character as a literal on a future loop
			continue
		}
		litLen++
	}
	// If there's anything left in "a" at this point then it's trailing literal
	// characters.
	if len(a) > 0 {
		to.WriteString(a)
	}
}

func posixShellSingleQuoted(a string, to *strings.Builder) {
	// Inside single quotes the only thing we need to escape are single
	// quotes themselves, which we achieve by temporarily leaving our
	// quotes and emitting a backslash escape, like: '\''

	to.WriteByte('\'')
	for {
		quot := strings.Index(a, "'")
		if quot == -1 { // No quotes left, so everything else is literal and we're done,
			to.WriteString(a)
			break
		}
		to.WriteString(a[:quot])
		to.WriteString(`'\''`)
		a = a[quot+1:]
	}
	to.WriteByte('\'')
}
