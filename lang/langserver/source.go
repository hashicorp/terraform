package langserver

import (
	"bufio"

	"github.com/hashicorp/hcl/v2"
)

type sourceLine struct {
	content []byte
	rng     hcl.Range
}

type sourceLines []sourceLine

func makeSourceLines(filename string, s []byte) sourceLines {
	var ret sourceLines
	sc := hcl.NewRangeScanner(s, filename, bufio.ScanLines)
	for sc.Scan() {
		ret = append(ret, sourceLine{
			content: sc.Bytes(),
			rng:     sc.Range(),
		})
	}
	if len(ret) == 0 {
		ret = append(ret, sourceLine{
			content: nil,
			rng: hcl.Range{
				Filename: filename,
				Start:    hcl.Pos{Line: 1, Column: 1},
				End:      hcl.Pos{Line: 1, Column: 1},
			},
		})
	}
	return ret
}

// byteOffsetToHCL converts a byte offset within a file into the equivalent
// position in HCL's representation.
func (ls sourceLines) byteOffsetToHCL(byte int) hcl.Pos {
	if len(ls) == 0 {
		return hcl.Pos{Line: 1, Column: 1, Byte: 0}
	}

	for i, srcLine := range ls {
		if srcLine.rng.ContainsOffset(byte) {
			lineNum := i + 1
			column := byte - srcLine.rng.Start.Byte

			return hcl.Pos{Line: lineNum, Column: column, Byte: byte}
		}
	}

	return hcl.Pos{Line: 1, Column: 1, Byte: 0}
}
