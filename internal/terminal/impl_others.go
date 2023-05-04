// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build !windows
// +build !windows

package terminal

import (
	"os"

	"golang.org/x/term"
)

// This is the implementation for all operating systems except Windows, where
// we don't expect to need to do any special initialization to get a working
// Virtual Terminal.
//
// For this implementation we just delegate everything upstream to
// golang.org/x/term, since it already has a variety of different
// implementations for quirks of more esoteric operating systems like plan9,
// and will hopefully grow to include others as Go is ported to other platforms
// in future.
//
// For operating systems that golang.org/x/term doesn't support either, it
// defaults to indicating that nothing is a terminal and returns an error when
// asked for a size, which we'll handle below.

func configureOutputHandle(f *os.File) (*OutputStream, error) {
	return &OutputStream{
		File:       f,
		isTerminal: isTerminalGolangXTerm,
		getColumns: getColumnsGolangXTerm,
	}, nil
}

func configureInputHandle(f *os.File) (*InputStream, error) {
	return &InputStream{
		File:       f,
		isTerminal: isTerminalGolangXTerm,
	}, nil
}

func isTerminalGolangXTerm(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func getColumnsGolangXTerm(f *os.File) int {
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		// Suggests that it's either not a terminal at all or that we're on
		// a platform that golang.org/x/term doesn't support. In both cases
		// we'll just return the placeholder default value.
		return defaultColumns
	}
	return width
}
