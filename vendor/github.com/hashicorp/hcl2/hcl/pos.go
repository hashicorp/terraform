package hcl

import "fmt"

// Pos represents a single position in a source file, by addressing the
// start byte of a unicode character encoded in UTF-8.
//
// Pos is generally used only in the context of a Range, which then defines
// which source file the position is within.
type Pos struct {
	// Line is the source code line where this position points. Lines are
	// counted starting at 1 and incremented for each newline character
	// encountered.
	Line int

	// Column is the source code column where this position points, in
	// unicode characters, with counting starting at 1.
	//
	// Column counts characters as they appear visually, so for example a
	// latin letter with a combining diacritic mark counts as one character.
	// This is intended for rendering visual markers against source code in
	// contexts where these diacritics would be rendered in a single character
	// cell. Technically speaking, Column is counting grapheme clusters as
	// used in unicode normalization.
	Column int

	// Byte is the byte offset into the file where the indicated character
	// begins. This is a zero-based offset to the first byte of the first
	// UTF-8 codepoint sequence in the character, and thus gives a position
	// that can be resolved _without_ awareness of Unicode characters.
	Byte int
}

// Range represents a span of characters between two positions in a source
// file.
//
// This struct is usually used by value in types that represent AST nodes,
// but by pointer in types that refer to the positions of other objects,
// such as in diagnostics.
type Range struct {
	// Filename is the name of the file into which this range's positions
	// point.
	Filename string

	// Start and End represent the bounds of this range. Start is inclusive
	// and End is exclusive.
	Start, End Pos
}

// RangeBetween returns a new range that spans from the beginning of the
// start range to the end of the end range.
//
// The result is meaningless if the two ranges do not belong to the same
// source file or if the end range appears before the start range.
func RangeBetween(start, end Range) Range {
	return Range{
		Filename: start.Filename,
		Start:    start.Start,
		End:      end.End,
	}
}

// ContainsOffset returns true if and only if the given byte offset is within
// the receiving Range.
func (r Range) ContainsOffset(offset int) bool {
	return offset >= r.Start.Byte && offset < r.End.Byte
}

// Ptr returns a pointer to a copy of the receiver. This is a convenience when
// ranges in places where pointers are required, such as in Diagnostic, but
// the range in question is returned from a method. Go would otherwise not
// allow one to take the address of a function call.
func (r Range) Ptr() *Range {
	return &r
}

// String returns a compact string representation of the receiver.
// Callers should generally prefer to present a range more visually,
// e.g. via markers directly on the relevant portion of source code.
func (r Range) String() string {
	if r.Start.Line == r.End.Line {
		return fmt.Sprintf(
			"%s:%d,%d-%d",
			r.Filename,
			r.Start.Line, r.Start.Column,
			r.End.Column,
		)
	} else {
		return fmt.Sprintf(
			"%s:%d,%d-%d,%d",
			r.Filename,
			r.Start.Line, r.Start.Column,
			r.End.Line, r.End.Column,
		)
	}
}
