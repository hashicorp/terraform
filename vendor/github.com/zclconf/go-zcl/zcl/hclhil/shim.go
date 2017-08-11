package hclhil

import (
	"github.com/zclconf/go-zcl/zcl"
	hclparser "github.com/hashicorp/hcl/hcl/parser"
	hcltoken "github.com/hashicorp/hcl/hcl/token"
	hilast "github.com/hashicorp/hil/ast"
	hilparser "github.com/hashicorp/hil/parser"
)

// errorRange attempts to extract a source range from the given error,
// returning a pointer to the range if possible or nil if not.
//
// errorRange understands HCL's "PosError" type, which wraps an error
// with a source position.
func errorRange(err error) *zcl.Range {
	switch terr := err.(type) {
	case *hclparser.PosError:
		rng := rangeFromHCLPos(terr.Pos)
		return &rng
	case *hilparser.ParseError:
		rng := rangeFromHILPos(terr.Pos)
		return &rng
	default:
		return nil
	}
}

func rangeFromHCLPos(pos hcltoken.Pos) zcl.Range {
	// HCL only marks single positions rather than ranges, so we adapt this
	// by creating a single-character range at the given position.
	return zcl.Range{
		Filename: pos.Filename,
		Start: zcl.Pos{
			Byte:   pos.Offset,
			Line:   pos.Line,
			Column: pos.Column,
		},
		End: zcl.Pos{
			Byte:   pos.Offset + 1,
			Line:   pos.Line,
			Column: pos.Column + 1,
		},
	}
}

func rangeFromHILPos(pos hilast.Pos) zcl.Range {
	// HIL only marks single positions rather than ranges, so we adapt this
	// by creating a single-character range at the given position.
	// HIL also doesn't track byte offsets, so we will hard-code these to
	// zero so that no position can be considered to be "inside" these
	// from a byte offset perspective.
	return zcl.Range{
		Filename: pos.Filename,
		Start: zcl.Pos{
			Byte:   0,
			Line:   pos.Line,
			Column: pos.Column,
		},
		End: zcl.Pos{
			Byte:   0,
			Line:   pos.Line,
			Column: pos.Column + 1,
		},
	}

}
