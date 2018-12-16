package langserver

import (
	"bufio"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/apparentlymart/go-textseg/textseg"
	"github.com/hashicorp/hcl/v2"

	lsp "github.com/hashicorp/terraform/internal/lsp"
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

// rangeHCLToLSP converts a range in HCL's representation to the equivalent
// position in LSP's.
func (ls sourceLines) rangeHCLToLSP(in hcl.Range) lsp.Range {
	return lsp.Range{
		Start: ls.posHCLToLSP(in.Start),
		End:   ls.posHCLToLSP(in.End),
	}
}

// posLSPToHCL converts a position in LSP's representation into the equivalent
// position in HCL's representation. If the given position is not within the
// content then the result is undefined.
func (ls sourceLines) posLSPToHCL(in lsp.Position) hcl.Pos {
	if len(ls) == 0 {
		return hcl.Pos{Line: 1, Column: 1, Byte: 0}
	}
	lspLine := int(in.Line)
	if lspLine >= len(ls) {
		return ls[len(ls)-1].rng.End
	}
	if lspLine < 0 {
		return ls[0].rng.Start
	}

	l := ls[lspLine]
	return l.posForLSPColumn(in.Character)
}

// posHCLToLSP converts a position in HCL's representation into the equivalent
// position in the LSP's representation. If the given position is not within
// the content then the result is undefined. The result is also undefined
// if the given range has a Byte value inconsistent with its Line and Column
// values.
func (ls sourceLines) posHCLToLSP(in hcl.Pos) lsp.Position {
	if len(ls) == 0 {
		return lsp.Position{Line: 0, Character: 0}
	}
	if (in.Line - 1) >= len(ls) {
		in = ls[len(ls)-1].rng.End
	} else if in.Line < 1 {
		in = ls[0].rng.Start
	}

	l := ls[in.Line-1]
	return lsp.Position{
		Line:      float64(in.Line - 1),
		Character: l.lspColumnForPos(in),
	}
}

// posLSPtoByte converts a position in LSP's representation into a byte
// offset into the full source buffer. If the given position is not within
// the content then the result is undefined. This is different than the
// byte offset returned in posLSPToHCL because it can potentially point into
// the middle of a grapheme cluster. It will produce an incorrect result if
// given a position referring to the second unit of a utf-16 surrogate pair.
//
// This should NOT be used to process incoming text change requests from the
// LSP client because the user may type into the middle of a surrogate pair
// and the rounding behavior of this method would cause the first unit
// to be lost, causing us to get out of sync with the client.
func (ls sourceLines) posLSPToByte(in lsp.Position) int {
	if len(ls) == 0 {
		return 0
	}
	lspLine := int(in.Line)
	if lspLine >= len(ls) {
		return ls[len(ls)-1].rng.End.Byte
	}
	if lspLine < 0 {
		return ls[0].rng.Start.Byte
	}

	l := ls[lspLine]
	return l.byteForLSPColumn(in.Character)
}

// allASCII returns true if the receiver is provably all ASCII, which allows
// for some fast paths where we can treat columns and bytes as equivalent.
func (l sourceLine) allASCII() bool {
	// If we have the same number of columns as bytes then our content is
	// all ASCII, since it clearly contains no multi-byte grapheme clusters.
	bytes := l.rng.End.Byte - l.rng.Start.Byte
	columns := l.rng.End.Column - l.rng.Start.Column
	return bytes == columns
}

// lspLen returns the length of the content of this line in characters as the
// LSP thinks of them, which is by counting how many code units would represent
// this string in UTF-16.
func (l sourceLine) lspLen() int {
	if l.allASCII() {
		// Easy path: length is the byte length
		return len(l.content)
	}

	chars := 0
	remain := l.content
	for len(remain) > 0 {
		r, l := utf8.DecodeRune(remain)
		remain = remain[l:]
		if r1, r2 := utf16.EncodeRune(r); r1 == 0xfffd && r2 == 0xfffd {
			chars++ // only one code unit needed for this one
		} else {
			chars += 2 // needs a surrogate pair
		}
	}
	return chars
}

// posForLSPColumn takes an lsp.Position.Character value for the receving line
// and finds the equivalent hcl.Pos for it.
func (l sourceLine) posForLSPColumn(lspCol float64) hcl.Pos {
	inCol := int(lspCol)
	if inCol < 0 {
		return l.rng.Start
	}

	// Easy path: if the entire line is ASCII then column counts are equivalent
	// in LSP vs. HCL aside from zero- vs. one-based counting.
	if l.allASCII() {
		return hcl.Pos{
			Line:   l.rng.Start.Line,
			Column: inCol + 1,
			Byte:   l.rng.Start.Byte + inCol,
		}
	}

	// If there are non-ASCII characters then we need to edge carefully
	// along the line while counting UTF-16 code units in our UTF-8 buffer,
	// since LSP columns are a count of UTF-16 units.
	byteCt := 0
	utf16Ct := 0
	colIdx := 1
	remain := l.content
	for {
		if len(remain) == 0 { // ran out of characters on the line, so given column is invalid
			return l.rng.End
		}
		if utf16Ct >= inCol { // we've found it
			return hcl.Pos{
				Line:   l.rng.Start.Line,
				Column: colIdx,
				Byte:   l.rng.Start.Byte + byteCt,
			}
		}
		adv, chBytes, _ := textseg.ScanGraphemeClusters(remain, true)
		remain = remain[adv:]
		byteCt += adv
		colIdx++
		for len(chBytes) > 0 {
			r, l := utf8.DecodeRune(chBytes)
			chBytes = chBytes[l:]
			c1, c2 := utf16.EncodeRune(r)
			if c1 == 0xfffd && c2 == 0xfffd {
				utf16Ct++ // codepoint fits in one 16-bit unit
			} else {
				utf16Ct += 2 // codepoint requires a surrogate pair
			}
		}
	}
}

// lspColumnForPos takes a hcl.Pos that must be within the receving line
// and returns its corresponding LSP column offset within the same line.
func (l sourceLine) lspColumnForPos(pos hcl.Pos) float64 {
	if pos.Column < l.rng.Start.Column || pos.Byte < l.rng.Start.Byte {
		return float64(l.rng.Start.Column - 1)
	} else if pos.Column > l.rng.End.Column || pos.Byte > l.rng.End.Byte {
		return float64(l.rng.End.Column - 1)
	}

	// Easy path: if the entire line is ASCII then column counts are equivalent
	// in LSP vs. HCL aside from zero- vs. one-based counting.
	if l.allASCII() {
		return float64(pos.Column - 1)
	}

	// If there are non-ASCII characters then we need to edge carefully
	// along the line while counting UTF-16 code units in our UTF-8 buffer,
	// since LSP columns are a count of UTF-16 units.
	utf16Ct := 0
	colIdx := 1
	remain := l.content
	for {
		if len(remain) == 0 { // ran out of characters on the line, so given position is invalid
			return float64(l.rng.End.Column - 1)
		}
		if colIdx >= pos.Column { // we've found it
			return float64(utf16Ct)
		}
		adv, chBytes, _ := textseg.ScanGraphemeClusters(remain, true)
		remain = remain[adv:]
		colIdx++
		for len(chBytes) > 0 {
			r, l := utf8.DecodeRune(chBytes)
			chBytes = chBytes[l:]
			c1, c2 := utf16.EncodeRune(r)
			if c1 == 0xfffd && c2 == 0xfffd {
				utf16Ct++ // codepoint fits in one 16-bit unit
			} else {
				utf16Ct += 2 // codepoint requires a surrogate pair
			}
		}
	}
}

// byteForLSPColumn takes an lsp.Position.Character value for the receving line
// and finds the byte offset of the start of the UTF-8 sequence that represents
// it in the overall source buffer. This is different than the byte returned
// by posForLSPColumn because it can return offsets that are partway through
// a grapheme cluster, while HCL positions always round to the nearest
// grapheme cluster.
//
// Note that even this can't produce an exact result; if the column index
// refers to the second unit of a UTF-16 surrogate pair then it is rounded
// down the first unit because UTF-8 sequences are not divisible in the same
// way.
func (l sourceLine) byteForLSPColumn(lspCol float64) int {
	inCol := int(lspCol)
	if inCol < 0 {
		return l.rng.Start.Byte
	}

	// Easy path: if the entire line is ASCII then column counts are equivalent
	// in LSP vs. HCL aside from zero- vs. one-based counting.
	if l.allASCII() {
		return l.rng.Start.Byte + inCol
	}

	// If there are non-ASCII characters then we need to edge carefully
	// along the line while counting UTF-16 code units in our UTF-8 buffer,
	// since LSP columns are a count of UTF-16 units.
	byteCt := 0
	utf16Ct := 0
	colIdx := 1
	remain := l.content
	for {
		if len(remain) == 0 { // ran out of characters on the line, so given column is invalid
			return l.rng.End.Byte
		}
		if utf16Ct >= inCol { // we've found it
			return l.rng.Start.Byte + byteCt
		}
		// Unlike our other conversion functions we're intentionally using
		// individual UTF-8 sequences here rather than grapheme clusters because
		// an LSP position might point into the middle of a grapheme cluster.

		adv, chBytes, _ := textseg.ScanUTF8Sequences(remain, true)
		remain = remain[adv:]
		byteCt += adv
		colIdx++
		for len(chBytes) > 0 {
			r, l := utf8.DecodeRune(chBytes)
			chBytes = chBytes[l:]
			c1, c2 := utf16.EncodeRune(r)
			if c1 == 0xfffd && c2 == 0xfffd {
				utf16Ct++ // codepoint fits in one 16-bit unit
			} else {
				utf16Ct += 2 // codepoint requires a surrogate pair
			}
		}
	}
}
