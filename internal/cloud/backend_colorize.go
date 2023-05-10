// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"regexp"

	"github.com/mitchellh/colorstring"
)

// TODO SvH: This file should be deleted and the type cliColorize should be
// renamed back to Colorize as soon as we can pass -no-color to the backend.

// colorsRe is used to find ANSI escaped color codes.
var colorsRe = regexp.MustCompile("\033\\[\\d{1,3}m")

// Colorer is the interface that must be implemented to colorize strings.
type Colorer interface {
	Color(v string) string
}

// Colorize is used to print output when the -no-color flag is used. It will
// strip all ANSI escaped color codes which are set while the operation was
// executed in Terraform Enterprise.
//
// When Terraform Enterprise supports run specific variables, this code can be
// removed as we can then pass the CLI flag to the backend and prevent the color
// codes from being written to the output.
type Colorize struct {
	cliColor *colorstring.Colorize
}

// Color will strip all ANSI escaped color codes and return a uncolored string.
func (c *Colorize) Color(v string) string {
	return colorsRe.ReplaceAllString(c.cliColor.Color(v), "")
}

// Colorize returns the Colorize structure that can be used for colorizing
// output. This is guaranteed to always return a non-nil value and so is useful
// as a helper to wrap any potentially colored strings.
func (b *Cloud) Colorize() Colorer {
	if b.CLIColor != nil && !b.CLIColor.Disable {
		return b.CLIColor
	}
	if b.CLIColor != nil {
		return &Colorize{cliColor: b.CLIColor}
	}
	return &Colorize{cliColor: &colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: true,
	}}
}
