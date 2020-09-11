package complete

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Args describes command line arguments
type Args struct {
	// All lists of all arguments in command line (not including the command itself)
	All []string
	// Completed lists of all completed arguments in command line,
	// If the last one is still being typed - no space after it,
	// it won't appear in this list of arguments.
	Completed []string
	// Last argument in command line, the one being typed, if the last
	// character in the command line is a space, this argument will be empty,
	// otherwise this would be the last word.
	Last string
	// LastCompleted is the last argument that was fully typed.
	// If the last character in the command line is space, this would be the
	// last word, otherwise, it would be the word before that.
	LastCompleted string
}

// Directory gives the directory of the current written
// last argument if it represents a file name being written.
// in case that it is not, we fall back to the current directory.
//
// Deprecated.
func (a Args) Directory() string {
	if info, err := os.Stat(a.Last); err == nil && info.IsDir() {
		return fixPathForm(a.Last, a.Last)
	}
	dir := filepath.Dir(a.Last)
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return "./"
	}
	return fixPathForm(a.Last, dir)
}

func newArgs(line string) Args {
	var (
		all       []string
		completed []string
	)
	parts := splitFields(line)
	if len(parts) > 0 {
		all = parts[1:]
		completed = removeLast(parts[1:])
	}
	return Args{
		All:           all,
		Completed:     completed,
		Last:          last(parts),
		LastCompleted: last(completed),
	}
}

// splitFields returns a list of fields from the given command line.
// If the last character is space, it appends an empty field in the end
// indicating that the field before it was completed.
// If the last field is of the form "a=b", it splits it to two fields: "a", "b",
// So it can be completed.
func splitFields(line string) []string {
	parts := strings.Fields(line)

	// Add empty field if the last field was completed.
	if len(line) > 0 && unicode.IsSpace(rune(line[len(line)-1])) {
		parts = append(parts, "")
	}

	// Treat the last field if it is of the form "a=b"
	parts = splitLastEqual(parts)
	return parts
}

func splitLastEqual(line []string) []string {
	if len(line) == 0 {
		return line
	}
	parts := strings.Split(line[len(line)-1], "=")
	return append(line[:len(line)-1], parts...)
}

// from returns a copy of Args of all arguments after the i'th argument.
func (a Args) from(i int) Args {
	if i >= len(a.All) {
		i = len(a.All) - 1
	}
	a.All = a.All[i+1:]

	if i >= len(a.Completed) {
		i = len(a.Completed) - 1
	}
	a.Completed = a.Completed[i+1:]
	return a
}

func removeLast(a []string) []string {
	if len(a) > 0 {
		return a[:len(a)-1]
	}
	return a
}

func last(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[len(args)-1]
}
