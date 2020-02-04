package command

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/command/cliconfig"
	oauthserver "github.com/hashicorp/terraform/command/testdata/login-oauth-server"
	"github.com/hashicorp/terraform/command/webbrowser"
	"github.com/hashicorp/terraform/httpclient"
	"github.com/hashicorp/terraform/version"
)

func TestLogin(t *testing.T) {
	// oauthserver.Handler is a stub OAuth server implementation that will,
	// on success, always issue a bearer token named "good-token".
	s := httptest.NewServer(oauthserver.Handler)
	defer s.Close()

	loginTestCase := func(test func(t *testing.T, c *LoginCommand, ui *cli.MockUi, inp func(string))) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			workDir, err := ioutil.TempDir("", "terraform-test-command-login")
			if err != nil {
				t.Fatalf("cannot create temporary directory: %s", err)
			}
			defer os.RemoveAll(workDir)

			// We'll use this context to avoid asynchronous tasks outliving
			// a single test run.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			ui := cli.NewMockUi()
			browserLauncher := webbrowser.NewMockLauncher(ctx)
			creds := cliconfig.EmptyCredentialsSourceForTests(filepath.Join(workDir, "credentials.tfrc.json"))
			svcs := disco.NewWithCredentialsSource(creds)
			svcs.SetUserAgent(httpclient.TerraformUserAgent(version.String()))

			inputBuf := &bytes.Buffer{}
			ui.InputReader = inputBuf

			svcs.ForceHostServices(svchost.Hostname("app.terraform.io"), map[string]interface{}{
				"login.v1": map[string]interface{}{
					// On app.terraform.io we use password-based authorization.
					// That's the only hostname that it's permitted for, so we can't
					// use a fake hostname here.
					"client":      "terraformcli",
					"token":       s.URL + "/token",
					"grant_types": []interface{}{"password"},
				},
			})
			svcs.ForceHostServices(svchost.Hostname("example.com"), map[string]interface{}{
				"login.v1": map[string]interface{}{
					// For this fake hostname we'll use a conventional OAuth flow,
					// with browser-based consent that we'll mock away using a
					// mock browser launcher below.
					"client": "anything-goes",
					"authz":  s.URL + "/authz",
					"token":  s.URL + "/token",
				},
			})
			svcs.ForceHostServices(svchost.Hostname("unsupported.example.net"), map[string]interface{}{
				// This host intentionally left blank.
			})

			c := &LoginCommand{
				Meta: Meta{
					Ui:              ui,
					BrowserLauncher: browserLauncher,
					Services:        svcs,
				},
			}

			test(t, c, ui, func(data string) {
				t.Helper()
				inputBuf.WriteString(data)
			})
		}
	}

	t.Run("defaulting to app.terraform.io with password flow", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi, inp func(string)) {
		// Enter "yes" at the consent prompt, then a username and then a password.
		inp("yes\nfoo\nbar\n")
		status := c.Run(nil)
		if status != 0 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		credsSrc := c.Services.CredentialsSource()
		creds, err := credsSrc.ForHost(svchost.Hostname("app.terraform.io"))
		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}
		if got, want := creds.Token(), "good-token"; got != want {
			t.Errorf("wrong token %q; want %q", got, want)
		}
	}))

	t.Run("example.com with authorization code flow", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi, inp func(string)) {
		// Enter "yes" at the consent prompt.
		inp("yes\n")
		status := c.Run([]string{"example.com"})
		if status != 0 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		credsSrc := c.Services.CredentialsSource()
		creds, err := credsSrc.ForHost(svchost.Hostname("example.com"))
		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}
		if got, want := creds.Token(), "good-token"; got != want {
			t.Errorf("wrong token %q; want %q", got, want)
		}
	}))

	t.Run("host without login support", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi, inp func(string)) {
		status := c.Run([]string{"unsupported.example.net"})
		if status == 0 {
			t.Fatalf("successful exit; want error")
		}

		if got, want := ui.ErrorWriter.String(), "Error: Host does not support Terraform login"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	}))
}
