package tfconfig

import (
	legacyhcltoken "github.com/hashicorp/hcl/hcl/token"
	"github.com/hashicorp/hcl2/hcl"
)

// SourcePos is a pointer to a particular location in a source file.
//
// This type is embedded into other structs to allow callers to locate the
// definition of each described module element. The SourcePos of an element
// is usually the first line of its definition, although the definition can
// be a little "fuzzy" with JSON-based config files.
type SourcePos struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

func sourcePos(filename string, line int) SourcePos {
	return SourcePos{
		Filename: filename,
		Line:     line,
	}
}

func sourcePosHCL(rng hcl.Range) SourcePos {
	// We intentionally throw away the column information here because
	// current and legacy HCL both disagree on the definition of a column
	// and so a line-only reference is the best granularity we can do
	// such that the result is consistent between both parsers.
	return SourcePos{
		Filename: rng.Filename,
		Line:     rng.Start.Line,
	}
}

func sourcePosLegacyHCL(pos legacyhcltoken.Pos, filename string) SourcePos {
	useFilename := pos.Filename
	// We'll try to use the filename given in legacy HCL position, but
	// in practice there's no way to actually get this populated via
	// the HCL API so it's usually empty except in some specialized
	// situations, such as positions in error objects.
	if useFilename == "" {
		useFilename = filename
	}
	return SourcePos{
		Filename: useFilename,
		Line:     pos.Line,
	}
}
