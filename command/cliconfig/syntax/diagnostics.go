package syntax

import (
	"fmt"

	hcl1parser "github.com/hashicorp/hcl/hcl/parser"
	hcl1token "github.com/hashicorp/hcl/hcl/token"
	hcl2 "github.com/hashicorp/hcl/v2"
)

// hcl1ErrorAsDiagnostic converts the given error, assumed to be returned from
// the HCL 1 parser or decoder, into an HCL 2 diagnostic that is as high-quality
// as possible given the limitations of HCL 1's error reporting.
//
// Not all HCL 1 errors carry accurate position information, so the caller must
// provide a default position to use for errors that lack one of their own.
// This default position should have a filename, which might need to be added
// using hcl1PosDefaultFilename before calling.
func hcl1ErrorAsDiagnostic(err error, defaultPos hcl1token.Pos) *hcl2.Diagnostic {
	switch err := err.(type) {
	case *hcl1parser.PosError:
		pos := hcl1PosDefaultFilename(err.Pos, defaultPos.Filename)
		return &hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid CLI configuration",
			Detail:   fmt.Sprintf("Error while processing the CLI configuration: %s.", err),
			Subject:  hcl1PosAsHCL2Range(pos).Ptr(),
		}
	default:
		return &hcl2.Diagnostic{
			Severity: hcl2.DiagError,
			Summary:  "Invalid CLI configuration",
			Detail:   fmt.Sprintf("Error while processing the CLI configuration: %s.", err),
			Subject:  hcl1PosAsHCL2Range(defaultPos).Ptr(),
		}
	}
}

// hcl1ErrorAsDiagnostics is a helper wrapper around hcl1ErrorAsDiagnostic that
// also wraps the result in a single-element HCL 2 Diagnostics, to ease the
// common case where a single HCL 1 error terminates all further processing.
func hcl1ErrorAsDiagnostics(err error, defaultPos hcl1token.Pos) hcl2.Diagnostics {
	return hcl2.Diagnostics{
		hcl1ErrorAsDiagnostic(err, defaultPos),
	}
}

// hcl1PosDefaultFilename inserts a default filename into the given position
// and returns it, unless the position already has a filename. HCL 1 rarely
// generates filenames in practice, so this is often necessary.
func hcl1PosDefaultFilename(given hcl1token.Pos, defaultFilename string) hcl1token.Pos {
	if given.Filename == "" {
		given.Filename = defaultFilename
	}
	return given
}

// hcl1PosAsHCL2Range converts the given HCL 1 position into an HCL 2 range.
// The given position should have a filename, which might require preprocessing
// it with hcl1PosDefaultFilename.
func hcl1PosAsHCL2Range(given hcl1token.Pos) hcl2.Range {
	// A single-byte/character range starting at the given position, just so
	// there's something for the recipient of the diagnostic to highlight as
	// the error.
	return hcl2.Range{
		Filename: given.Filename,
		Start: hcl2.Pos{
			Line:   given.Line,
			Column: given.Column, // Note: will be incorrect in the presence of multi-rune characters, cause HCL 1 has a rune-based idea of columns.
			Byte:   given.Offset,
		},
		End: hcl2.Pos{
			Line:   given.Line,
			Column: given.Column + 1,
			Byte:   given.Offset + 1,
		},
	}
}
