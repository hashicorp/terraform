// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestStateListHuman_List(t *testing.T) {
	cases := map[string]struct {
		addresses      []string
		diags          tfdiags.Diagnostics
		expectedStdout string
	}{
		"basic": {
			addresses: []string{
				"aws_instance.one",
				"aws_instance.two",
				"aws_instance.three",
			},
			expectedStdout: "aws_instance.one\naws_instance.two\naws_instance.three",
		},
		"with diags": {
			addresses: []string{
				"aws_instance.one",
				"aws_instance.two",
				"aws_instance.three",
			},
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning.",
					"This is an example warning detail.",
				),
			},
			expectedStdout: "aws_instance.one\naws_instance.two\naws_instance.three\n\n" +
				"Warning: Example warning.\n\nThis is an example warning detail.",
		},
	}
	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewStateList(arguments.ViewHuman, view)

			v.List(tc.addresses, tc.diags)

			output := done(t)
			if diff := cmp.Diff(tc.expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
				t.Fatalf("unexpected stdout diff:\n%s", diff)
			}
		})
	}
}

func TestStateListHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateList(arguments.ViewHuman, view)

	diags := tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Error,
			"Example error.",
			"This is an example error detail.",
		),
	}
	v.Diagnostics(diags)

	output := done(t)
	expectedStdout := ""
	if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
		t.Fatalf("unexpected stdout diff:\n%s", diff)
	}
	expectedStderr := "Error: Example error.\n\nThis is an example error detail."
	if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
		t.Fatalf("unexpected stderr diff:\n%s", diff)
	}
}
