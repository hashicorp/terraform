// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package repl

import (
	"strings"
	"testing"
)

func TestExpressionEntryCouldContinue(t *testing.T) {
	tests := []struct {
		Input []string
		Want  bool
	}{
		{
			nil,
			false,
		},
		{
			[]string{
				"",
			},
			false,
		},
		{
			[]string{
				"foo(",
				"", // trailing newline forces termination
			},
			false,
		},

		// parens
		{
			[]string{
				"foo()",
			},
			false,
		},
		{
			[]string{
				"foo(",
			},
			true,
		},
		{
			[]string{
				"foo(",
				"  bar,",
				")",
			},
			false,
		},

		// brackets
		{
			[]string{
				"[]",
			},
			false,
		},
		{
			[]string{
				"[",
			},
			true,
		},
		{
			[]string{
				"[",
				"]",
			},
			false,
		},

		// braces
		{
			[]string{
				"{}",
			},
			false,
		},
		{
			[]string{
				"{",
			},
			true,
		},
		{
			[]string{
				"{",
				"}",
			},
			false,
		},

		// quotes
		// HCL doesn't allow splitting quoted strings over multiple lines, so
		// these never cause continuation. (Use heredocs instead for that)
		{
			[]string{
				`""`,
			},
			false,
		},
		{
			[]string{
				`"`,
			},
			false,
		},
		{
			[]string{
				`"`,
				`"`,
			},
			false,
		},

		// heredoc templates
		{
			[]string{
				`<<EOT`,
				`EOT`,
			},
			false,
		},
		{
			[]string{
				`<<EOT`,
			},
			true,
		},
		{
			[]string{
				`<<EOT`,
				`beep`,
				`EOT`,
			},
			false,
		},
		{
			[]string{
				`<<EOT`,
				`beep`,
			},
			true,
		},
		{
			[]string{
				`<<-EOT`,
				`EOT`,
			},
			false,
		},
		{
			[]string{
				`<<-EOT`,
			},
			true,
		},
		{
			[]string{
				`<<-EOT`,
				`beep`,
				`EOT`,
			},
			false,
		},
		{
			[]string{
				`<<-EOT`,
				`beep`,
			},
			true,
		},
		{
			// In the following it's actually the heredoc that's keeping the
			// newline sequence going, rather than the control sequence, but
			// this is here to test a reasonable combination of things someone
			// might enter.
			[]string{
				`<<EOT`,
				`%{ for x in y }`,
			},
			true,
		},
		{
			[]string{
				`<<EOT`,
				`%{ for x in y }`,
				`boop`,
				`%{ endfor }`,
			},
			true,
		},
		{
			[]string{
				`<<EOT`,
				`%{ for x in y }`,
				`boop`,
				`%{ endfor }`,
				`EOT`,
			},
			false,
		},
		{
			[]string{
				`<<EOT`,
				`]`, // literal bracket, so doesn't count as a mismatch
			},
			true,
		},

		// template interpolation/control inside quotes
		// although quotes alone cannot span over multiple lines, a
		// template sequence creates a nested context where newlines are
		// allowed.
		{
			[]string{
				`"${hello}"`,
			},
			false,
		},
		{
			[]string{
				`"${hello`,
				`}"`,
			},
			false,
		},
		{
			[]string{
				`"${`,
				`  hello`,
				`}"`,
			},
			false,
		},
		{
			[]string{
				`"${`,
				`  hello`,
			},
			true,
		},
		{
			[]string{
				`"%{ for x in y }%{ endfor }"`,
			},
			false,
		},
		{
			[]string{
				`"%{`,
				`   for x in y }%{ endfor }"`,
			},
			false,
		},
		{
			// This case returns false because the control sequence itself
			// ends before the newline, and quoted literals are not allowed
			// to contain newlines, so this is a parse error.
			[]string{
				`"%{ for x in y }`,
			},
			false,
		},

		// mismatched brackets
		// these combinations should always return false so that we can
		// report the syntax error immediately.
		{
			[]string{
				`([)`,
			},
			false,
		},
		{
			[]string{
				`"${]`,
			},
			false,
		},
		{
			[]string{
				`"%{]`,
			},
			false,
		},
	}

	for _, test := range tests {
		name := strings.Join(test.Input, "⮠")
		t.Run(name, func(t *testing.T) {
			got := ExpressionEntryCouldContinue(test.Input)
			if got != test.Want {
				t.Errorf(
					"wrong result\ninput:\n%s\ngot:  %t\nwant: %t",
					strings.Join(test.Input, "\n"),
					got, test.Want,
				)
			}
		})
	}
}
