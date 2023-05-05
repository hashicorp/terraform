// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terminal

import (
	"os"
)

const defaultColumns int = 78
const defaultIsTerminal bool = false

// OutputStream represents an output stream that might or might not be connected
// to a terminal.
//
// There are typically two instances of this: one representing stdout and one
// representing stderr.
type OutputStream struct {
	File *os.File

	// Interacting with a terminal is typically platform-specific, so we
	// factor out these into virtual functions, although we have default
	// behaviors suitable for non-Terminal output if any of these isn't
	// set. (We're using function pointers rather than interfaces for this
	// because it allows us to mix both normal methods and virtual methods
	// on the same type, without a bunch of extra complexity.)
	isTerminal func(*os.File) bool
	getColumns func(*os.File) int
}

// Columns returns a number of character cell columns that we expect will
// fill the width of the terminal that stdout is connected to, or a reasonable
// placeholder value of 78 if the output doesn't seem to be a terminal.
//
// This is a best-effort sort of function which may give an inaccurate result
// in various cases. For example, callers storing the result will not react
// to subsequent changes in the terminal width, and indeed this function itself
// may not be able to either, depending on the constraints of the current
// execution context.
func (s *OutputStream) Columns() int {
	if s.getColumns == nil {
		return defaultColumns
	}
	return s.getColumns(s.File)
}

// IsTerminal returns true if we expect that the stream is connected to a
// terminal which supports VT100-style formatting and cursor control sequences.
func (s *OutputStream) IsTerminal() bool {
	if s.isTerminal == nil {
		return defaultIsTerminal
	}
	return s.isTerminal(s.File)
}

// InputStream represents an input stream that might or might not be a terminal.
//
// There is typically only one instance of this type, representing stdin.
type InputStream struct {
	File *os.File

	// Interacting with a terminal is typically platform-specific, so we
	// factor out these into virtual functions, although we have default
	// behaviors suitable for non-Terminal output if any of these isn't
	// set. (We're using function pointers rather than interfaces for this
	// because it allows us to mix both normal methods and virtual methods
	// on the same type, without a bunch of extra complexity.)
	isTerminal func(*os.File) bool
}

// IsTerminal returns true if we expect that the stream is connected to a
// terminal which can support interactive input.
//
// If this returns false, callers might prefer to skip elaborate input prompt
// functionality like tab completion and instead just treat the input as a
// raw byte stream, or perhaps skip prompting for input at all depending on the
// situation.
func (s *InputStream) IsTerminal() bool {
	if s.isTerminal == nil {
		return defaultIsTerminal
	}
	return s.isTerminal(s.File)
}
