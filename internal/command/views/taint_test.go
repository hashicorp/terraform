// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestTaintHuman_LogResourceTainted(t *testing.T) {
	r := addrs.RootModuleInstance.ResourceInstance(
		addrs.ManagedResourceMode,
		"boop",
		"beep",
		addrs.NoKey,
	)

	cases := map[string]struct {
		resource       addrs.AbsResourceInstance
		diags          tfdiags.Diagnostics
		expectedStdout string
	}{
		"basic": {
			resource:       r,
			expectedStdout: "Resource instance boop.beep has been marked as tainted.",
		},
		"with warning": {
			resource: r,
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning.",
					"This is an example warning detail.",
				),
			},
			expectedStdout: "Warning: Example warning.\n\nThis is an example warning detail.\n" +
				"Resource instance boop.beep has been marked as tainted.",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewTaint(arguments.ViewHuman, view)

			v.LogResourceTainted(tc.resource, tc.diags)

			output := done(t)
			expectedStdout := tc.expectedStdout
			if diff := cmp.Diff(expectedStdout, strings.TrimSpace(output.Stdout())); diff != "" {
				t.Fatalf("unexpected stdout diff:\n%s", diff)
			}
			expectedStderr := ""
			if diff := cmp.Diff(expectedStderr, strings.TrimSpace(output.Stderr())); diff != "" {
				t.Fatalf("unexpected stderr diff:\n%s", diff)
			}
		})
	}
}

func TestTaintHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewTaint(arguments.ViewHuman, view)

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
