// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path/filepath"
	"strings"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"
	svcauth "github.com/hashicorp/terraform-svchost/auth"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
)

func TestLogout(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	ui := testUiWrapped(t)
	credsSrc := cliconfig.EmptyCredentialsSourceForTests(filepath.Join(workDir, "credentials.tfrc.json"))

	c := &LogoutCommand{
		Meta: Meta{
			Ui:       ui,
			Services: disco.NewWithCredentialsSource(credsSrc),
		},
	}

	testCases := []struct {
		// Hostname to associate a pre-stored token
		hostname string
		// Command-line arguments
		args []string
		// true iff the token at hostname should be removed by the command
		shouldRemove bool
	}{
		// If no command-line arguments given, should remove app.terraform.io token
		{"app.terraform.io", []string{}, true},

		// Can still specify app.terraform.io explicitly
		{"app.terraform.io", []string{"app.terraform.io"}, true},

		// Can remove tokens for other hostnames
		{"tfe.example.com", []string{"tfe.example.com"}, true},

		// Logout does not remove tokens for other hostnames
		{"tfe.example.com", []string{"other-tfe.acme.com"}, false},
	}
	for _, tc := range testCases {
		host := svchost.Hostname(tc.hostname)
		token := svcauth.HostCredentialsToken("some-token")
		err := credsSrc.StoreForHost(host, token)
		if err != nil {
			t.Fatalf("unexpected error storing credentials: %s", err)
		}

		status := c.Run(tc.args)
		if status != 0 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		creds, err := credsSrc.ForHost(host)
		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}
		if tc.shouldRemove {
			if creds != nil {
				t.Errorf("wrong token %q; should have no token", creds.Token())
			}
		} else {
			if got, want := creds.Token(), "some-token"; got != want {
				t.Errorf("wrong token %q; want %q", got, want)
			}
		}
	}
}

func TestLogout_argsInvalid(t *testing.T) {
	testCases := []struct {
		// Command-line arguments
		args               []string
		expectedErrSnippet string
	}{
		// Unrecognized flag
		{[]string{"-foobar"}, "Usage: terraform [global options] logout [hostname]"},

		// Extra positional argument flag
		{[]string{"app.terraform.io", "foobar"}, "Usage: terraform [global options] logout [hostname]"},
	}
	for _, tc := range testCases {

		workDir := t.TempDir()

		ui := testUiWrapped(t)
		credsSrc := cliconfig.EmptyCredentialsSourceForTests(filepath.Join(workDir, "credentials.tfrc.json"))

		c := &LogoutCommand{
			Meta: Meta{
				Ui:       ui,
				Services: disco.NewWithCredentialsSource(credsSrc),
			},
		}

		status := c.Run(tc.args)
		if status != 1 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		if !strings.Contains(ui.ErrorWriter.String(), tc.expectedErrSnippet) {
			t.Errorf("expected error snippet %q not found in stderr:\n%s", tc.expectedErrSnippet, ui.ErrorWriter.String())
		}
	}
}
