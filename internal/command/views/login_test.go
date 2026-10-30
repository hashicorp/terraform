// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestLoginHuman_ViaOAuthTokenGetterInterface(t *testing.T) {
	hostname := svchost.Hostname("example.com")
	url := "https://example.com/login"

	t.Run("OpeningLoginPage", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		view.Configure(&arguments.View{NoColor: true})
		v := NewLogin(arguments.ViewHuman, view)

		v.OpeningLoginPage(hostname, url)

		output := done(t)

		// Assert contents
		// This must be done separately for stdout and stderr due to
		// the interleaving of output caused in tests by (TestOutput).All()
		gotStdout := strings.TrimSpace(output.Stdout())
		wantStdout := strings.TrimSpace("Terraform must now open a web browser to the login page for example.com.\n\nIf a browser does not open this automatically, open the following URL to proceed:\n    https://example.com/login")
		if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
		gotStderr := strings.TrimSpace(output.Stderr())
		wantStderr := strings.TrimSpace("")
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
	})
	t.Run("PromptToOpenLoginPage", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		view.Configure(&arguments.View{NoColor: true})
		v := NewLogin(arguments.ViewHuman, view)

		v.PromptToOpenLoginPage(hostname, url)

		output := done(t)

		// Assert contents
		// This must be done separately for stdout and stderr due to
		// the interleaving of output caused in tests by (TestOutput).All()
		gotStdout := strings.TrimSpace(output.Stdout())
		wantStdout := strings.TrimSpace("Open the following URL to access the login page for example.com:\n    https://example.com/login")
		if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
		gotStderr := strings.TrimSpace(output.Stderr())
		wantStderr := strings.TrimSpace("")
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
	})
	t.Run("WaitingForLoginSuccess", func(t *testing.T) {
		streams, done := terminal.StreamsForTesting(t)
		view := NewView(streams)
		view.Configure(&arguments.View{NoColor: true})
		v := NewLogin(arguments.ViewHuman, view)

		v.WaitingForLoginSuccess()

		output := done(t)

		// Assert contents
		// This must be done separately for stdout and stderr due to
		// the interleaving of output caused in tests by (TestOutput).All()
		gotStdout := strings.TrimSpace(output.Stdout())
		wantStdout := strings.TrimSpace("Terraform will now wait for the host to signal that login was successful.")
		if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
		gotStderr := strings.TrimSpace(output.Stderr())
		wantStderr := strings.TrimSpace("")
		if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
			t.Fatalf("unexpected diff in human output:\n%s", diff)
		}
	})
}

func TestLoginHuman_Diagnostics(t *testing.T) {
	testCases := map[string]struct {
		diags      tfdiags.Diagnostics
		wantStdout string
		wantStderr string
	}{
		"error": {
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Example error",
					"This is an example error message.",
				),
			},
			"",
			"Error: Example error\n\nThis is an example error message.\n\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			view := NewView(streams)
			view.Configure(&arguments.View{NoColor: true})
			v := NewLogin(arguments.ViewHuman, view)

			v.Diagnostics(tc.diags)

			output := done(t)

			// Assert contents
			// This must be done separately for stdout and stderr due to
			// the interleaving of output caused in tests by (TestOutput).All()
			gotStdout := strings.TrimSpace(output.Stdout())
			wantStdout := strings.TrimSpace(tc.wantStdout)
			if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
			gotStderr := strings.TrimSpace(output.Stderr())
			wantStderr := strings.TrimSpace(tc.wantStderr)
			if diff := cmp.Diff(wantStderr, gotStderr); diff != "" {
				t.Fatalf("unexpected diff in human output:\n%s", diff)
			}
		})
	}
}
