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

func TestLogoutHuman_NoCredentialsStored(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewLogout(arguments.ViewHuman, view)

	host := "app.example.com"
	v.NoCredentialsStored(host)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace("No credentials for app.example.com are stored.")
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	if diff := cmp.Diff("", gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestLogoutHuman_NotifyTokenRemovedFromCredentialsHelper(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewLogout(arguments.ViewHuman, view)

	host := "app.example.com"
	helper := "foobar"
	v.NotifyTokenRemovedFromCredentialsHelper(host, helper)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace("Removing the stored credentials for app.example.com from the configured\n\"foobar\" credentials helper.")
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	if diff := cmp.Diff("", gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestLogoutHuman_NotifyTokenRemovedFromFile(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewLogout(arguments.ViewHuman, view)

	host := "app.example.com"
	file := "/path/to/credentials/file"
	v.NotifyTokenRemovedFromFile(host, file)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace("Removing the stored credentials for app.example.com from the following file:\n    /path/to/credentials/file")
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	if diff := cmp.Diff("", gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestLogoutHuman_LogoutSuccess(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewLogout(arguments.ViewHuman, view)

	host := "app.example.com"
	v.LogoutSuccess(host)

	output := done(t)

	// Assert contents
	// This must be done separately for stdout and stderr due to
	// the interleaving of output caused in tests by (TestOutput).All()
	gotStdout := strings.TrimSpace(output.Stdout())
	wantStdout := strings.TrimSpace("Success! Terraform has removed the stored API token for app.example.com.")
	if diff := cmp.Diff(wantStdout, gotStdout); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
	gotStderr := strings.TrimSpace(output.Stderr())
	if diff := cmp.Diff("", gotStderr); diff != "" {
		t.Fatalf("unexpected diff in human output:\n%s", diff)
	}
}

func TestLogoutHuman_Diagnostics(t *testing.T) {
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
			v := NewLogout(arguments.ViewHuman, view)

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
