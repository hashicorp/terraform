package tfdiags

import (
	"fmt"
	"os"
	"path/filepath"
)

type SourceRange struct {
	Filename   string
	Start, End SourcePos
}

type SourcePos struct {
	Line, Column, Byte int
}

// StartString returns a string representation of the start of the range,
// including the filename and the line and column numbers.
func (r SourceRange) StartString() string {
	filename := r.Filename

	// We'll try to relative-ize our filename here so it's less verbose
	// in the common case of being in the current working directory. If not,
	// we'll just show the full path.
	wd, err := os.Getwd()
	if err == nil {
		relFn, err := filepath.Rel(wd, filename)
		if err == nil {
			filename = relFn
		}
	}

	return fmt.Sprintf("%s:%d,%d", filename, r.Start.Line, r.Start.Column)
}
