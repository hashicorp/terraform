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

func TestUnlockHuman_UnlockSuccessful(t *testing.T) {
	message := "Terraform state has been successfully unlocked!\n\nThe state has been unlocked, and Terraform commands should now be able to\nobtain a new lock on the remote state."

	cases := map[string]struct {
		diags          tfdiags.Diagnostics
		expectedStdout string
	}{
		"basic": {
			diags:          tfdiags.Diagnostics{},
			expectedStdout: message,
		},
		"with warning": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning.",
					"This is an example warning detail.",
				),
			},
			expectedStdout: "Warning: Example warning.\n\nThis is an example warning detail.\n" +
				message,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewUnlock(arguments.ViewHuman, view)

			v.UnlockSuccessful(tc.diags)

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

func TestUnlockHuman_ForceUnlockCancelled(t *testing.T) {
	message := "Cancelled force-unlock."

	cases := map[string]struct {
		diags          tfdiags.Diagnostics
		expectedStdout string
	}{
		"basic": {
			diags:          tfdiags.Diagnostics{},
			expectedStdout: message,
		},
		"with warning": {
			diags: tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Warning,
					"Example warning.",
					"This is an example warning detail.",
				),
			},
			expectedStdout: "Warning: Example warning.\n\nThis is an example warning detail.\n" +
				message,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewUnlock(arguments.ViewHuman, view)

			v.ForceUnlockCancelled(tc.diags)

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

func TestUnlockHuman_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewUnlock(arguments.ViewHuman, view)

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
