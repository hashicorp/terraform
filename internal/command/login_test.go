package command

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	oauthserver "github.com/hashicorp/terraform/internal/command/testdata/login-oauth-server"
	tfeserver "github.com/hashicorp/terraform/internal/command/testdata/login-tfe-server"
	"github.com/hashicorp/terraform/internal/command/webbrowser"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/version"
)

func TestLogin(t *testing.T) {
	// oauthserver.Handler is a stub OAuth server implementation that will,
	// on success, always issue a bearer token named "good-token".
	s := httptest.NewServer(oauthserver.Handler)
	defer s.Close()

	// tfeserver.Handler is a stub TFE API implementation which will respond
	// to ping and current account requests, when requests are authenticated
	// with token "good-token"
	ts := httptest.NewServer(tfeserver.Handler)
	defer ts.Close()

	loginTestCase := func(test func(t *testing.T, c *LoginCommand, ui *cli.MockUi)) func(t *testing.T) {
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

			// Do not use the NewMockUi initializer here, as we want to delay
			// the call to init until after setting up the input mocks
			ui := new(cli.MockUi)

			browserLauncher := webbrowser.NewMockLauncher(ctx)
			creds := cliconfig.EmptyCredentialsSourceForTests(filepath.Join(workDir, "credentials.tfrc.json"))
			svcs := disco.NewWithCredentialsSource(creds)
			svcs.SetUserAgent(httpclient.TerraformUserAgent(version.String()))

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
			svcs.ForceHostServices(svchost.Hostname("with-scopes.example.com"), map[string]interface{}{
				"login.v1": map[string]interface{}{
					// with scopes
					// mock browser launcher below.
					"client": "scopes_test",
					"authz":  s.URL + "/authz",
					"token":  s.URL + "/token",
					"scopes": []interface{}{"app1.full_access", "app2.read_only"},
				},
			})
			svcs.ForceHostServices(svchost.Hostname("app.terraform.io"), map[string]interface{}{
				// This represents Terraform Cloud, which does not yet support the
				// login API, but does support its own bespoke tokens API.
				"tfe.v2":   ts.URL + "/api/v2",
				"tfe.v2.1": ts.URL + "/api/v2",
				"tfe.v2.2": ts.URL + "/api/v2",
				"motd.v1":  ts.URL + "/api/terraform/motd",
			})
			svcs.ForceHostServices(svchost.Hostname("tfe.acme.com"), map[string]interface{}{
				// This represents a Terraform Enterprise instance which does not
				// yet support the login API, but does support its own bespoke tokens API.
				"tfe.v2":   ts.URL + "/api/v2",
				"tfe.v2.1": ts.URL + "/api/v2",
				"tfe.v2.2": ts.URL + "/api/v2",
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

			test(t, c, ui)
		}
	}

	t.Run("app.terraform.io (no login support)", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "yes" at the consent prompt, then paste a token with some
		// accidental whitespace.
		defer testInputMap(t, map[string]string{
			"approve": "yes",
			"token":   "  good-token ",
		})()
		status := c.Run([]string{"app.terraform.io"})
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
		if got, want := ui.OutputWriter.String(), "Welcome to Terraform Cloud!"; !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, but was:\n%s", want, got)
		}
	}))

	t.Run("example.com with authorization code flow", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "yes" at the consent prompt.
		defer testInputMap(t, map[string]string{
			"approve": "yes",
		})()
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

		if got, want := ui.OutputWriter.String(), "Terraform has obtained and saved an API token."; !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, but was:\n%s", want, got)
		}
	}))

	t.Run("example.com results in no scopes", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {

		host, _ := c.Services.Discover("example.com")
		client, _ := host.ServiceOAuthClient("login.v1")
		if len(client.Scopes) != 0 {
			t.Errorf("unexpected scopes %q; expected none", client.Scopes)
		}
	}))

	t.Run("with-scopes.example.com with authorization code flow and scopes", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "yes" at the consent prompt.
		defer testInputMap(t, map[string]string{
			"approve": "yes",
		})()
		status := c.Run([]string{"with-scopes.example.com"})
		if status != 0 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		credsSrc := c.Services.CredentialsSource()
		creds, err := credsSrc.ForHost(svchost.Hostname("with-scopes.example.com"))

		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}

		if got, want := creds.Token(), "good-token"; got != want {
			t.Errorf("wrong token %q; want %q", got, want)
		}

		if got, want := ui.OutputWriter.String(), "Terraform has obtained and saved an API token."; !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, but was:\n%s", want, got)
		}
	}))

	t.Run("with-scopes.example.com results in expected scopes", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {

		host, _ := c.Services.Discover("with-scopes.example.com")
		client, _ := host.ServiceOAuthClient("login.v1")

		expectedScopes := [2]string{"app1.full_access", "app2.read_only"}

		var foundScopes [2]string
		copy(foundScopes[:], client.Scopes)

		if foundScopes != expectedScopes || len(client.Scopes) != len(expectedScopes) {
			t.Errorf("unexpected scopes %q; want %q", client.Scopes, expectedScopes)
		}
	}))

	t.Run("TFE host without login support", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "yes" at the consent prompt, then paste a token with some
		// accidental whitespace.
		defer testInputMap(t, map[string]string{
			"approve": "yes",
			"token":   "  good-token ",
		})()
		status := c.Run([]string{"tfe.acme.com"})
		if status != 0 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		credsSrc := c.Services.CredentialsSource()
		creds, err := credsSrc.ForHost(svchost.Hostname("tfe.acme.com"))
		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}
		if got, want := creds.Token(), "good-token"; got != want {
			t.Errorf("wrong token %q; want %q", got, want)
		}

		if got, want := ui.OutputWriter.String(), "Logged in to Terraform Enterprise"; !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, but was:\n%s", want, got)
		}
	}))

	t.Run("TFE host without login support, incorrectly pasted token", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "yes" at the consent prompt, then paste an invalid token.
		defer testInputMap(t, map[string]string{
			"approve": "yes",
			"token":   "good-tok",
		})()
		status := c.Run([]string{"tfe.acme.com"})
		if status != 1 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		credsSrc := c.Services.CredentialsSource()
		creds, err := credsSrc.ForHost(svchost.Hostname("tfe.acme.com"))
		if err != nil {
			t.Errorf("failed to retrieve credentials: %s", err)
		}
		if creds != nil {
			t.Errorf("wrong token %q; should have no token", creds.Token())
		}
	}))

	t.Run("host without login or TFE API support", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		status := c.Run([]string{"unsupported.example.net"})
		if status == 0 {
			t.Fatalf("successful exit; want error")
		}

		if got, want := ui.ErrorWriter.String(), "Error: Host does not support Terraform tokens API"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	}))

	t.Run("answering no cancels", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "no" at the consent prompt
		defer testInputMap(t, map[string]string{
			"approve": "no",
		})()
		status := c.Run(nil)
		if status != 1 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		if got, want := ui.ErrorWriter.String(), "Login cancelled"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	}))

	t.Run("answering y cancels", loginTestCase(func(t *testing.T, c *LoginCommand, ui *cli.MockUi) {
		// Enter "y" at the consent prompt
		defer testInputMap(t, map[string]string{
			"approve": "y",
		})()
		status := c.Run(nil)
		if status != 1 {
			t.Fatalf("unexpected error code %d\nstderr:\n%s", status, ui.ErrorWriter.String())
		}

		if got, want := ui.ErrorWriter.String(), "Login cancelled"; !strings.Contains(got, want) {
			t.Fatalf("missing expected error message\nwant: %s\nfull output:\n%s", want, got)
		}
	}))
}
