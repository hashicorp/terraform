// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package format

import (
	"strings"

	"github.com/mitchellh/colorstring"
	wordwrap "github.com/mitchellh/go-wordwrap"
)

// HorizontalRule returns a newline character followed by a number of
// horizontal line characters to fill the given width.
//
// If the given colorize has colors enabled, the rule will also be given a
// dark grey color to attempt to visually de-emphasize it for sighted users.
//
// This is intended for printing to the UI via mitchellh/cli.UI.Output, or
// similar, which will automatically append a trailing newline too.
func HorizontalRule(color *colorstring.Colorize, width int) string {
	if width <= 1 {
		return "\n"
	}
	rule := strings.Repeat("─", width-1)
	if color == nil { // sometimes unit tests don't populate this properly
		return "\n" + rule
	}
	return color.Color("[dark_gray]\n" + rule)
}

// WordWrap takes a string containing unbroken lines of text and inserts
// newline characters to try to make the text fit within the given width.
//
// The string can already contain newline characters, for example if you are
// trying to render multiple paragraphs of text. (In that case, our usual
// style would be to have _two_ newline characters as the paragraph separator.)
//
// As a special case, any line that begins with at least one space will be left
// unbroken. This allows including literal segments in the output, such as
// code snippets or filenames, where word wrapping would be confusing.
func WordWrap(str string, width int) string {
	if width <= 1 {
		// Silly edge case. We'll just return the original string to avoid
		// panicking or doing other weird stuff.
		return str
	}

	var buf strings.Builder
	lines := strings.Split(str, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, " ") {
			line = wordwrap.WrapString(line, uint(width-1))
		}
		if i > 0 {
			buf.WriteByte('\n') // reintroduce the newlines we skipped in Scan
		}
		buf.WriteString(line)
	}
	return buf.String()
}
