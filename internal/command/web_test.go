package command

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/command/webbrowser"
	"github.com/hashicorp/terraform/internal/command/webcommand"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/cli"
)

func TestWeb(t *testing.T) {
	t.Parallel()

	type TestDeps struct {
		Backend         *backendForTestWeb
		BrowserLauncher *webbrowser.MockLauncher
		UI              *cli.MockUi
		TestServerURL   string
	}

	newWebCommand := func(t *testing.T) (*WebCommand, TestDeps, func()) {
		ctx := context.Background()

		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("content-length", "0")
			w.Header().Set("content-type", "text/html")
			w.WriteHeader(200)
		}))

		testServerURL, err := url.Parse(testServer.URL)
		if err != nil {
			t.Fatalf("httptest.NewServer returned invalid URL: %s", err)
		}

		innerBackend := inmem.New().(*inmem.Backend)
		backend := &backendForTestWeb{
			Backend: innerBackend,
			retURL:  testServerURL,
		}
		browserLauncher := webbrowser.NewMockLauncher(ctx)
		ui := cli.NewMockUi()

		deps := TestDeps{
			Backend:         backend,
			BrowserLauncher: browserLauncher,
			UI:              ui,
			TestServerURL:   testServer.URL,
		}

		streams, closeStreams := terminal.StreamsForTesting(t)

		close := func() {
			closeStreams(t)
			testServer.Close()
		}

		return &WebCommand{
			Meta: Meta{
				Ui:                    ui,
				BrowserLauncher:       browserLauncher,
				FakeBackendForTesting: backend,
				View:                  views.NewView(streams),
			},
		}, deps, close
	}

	tests := map[string]struct {
		args             []string
		wantTargetObject webcommand.TargetObject
	}{
		"no options": {
			nil,
			webcommand.TargetObjectCurrentWorkspace,
		},
		"-latest-run": {
			[]string{"-latest-run"},
			webcommand.TargetObjectLatestRun,
		},
		"-run=foo": {
			[]string{"-run=foo"},
			webcommand.TargetObjectRun{RunID: "foo"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			c, deps, close := newWebCommand(t)
			defer close()

			result := c.Run(test.args)
			if result != 0 {
				t.Fatalf("failed; expected success")
			}
			deps.BrowserLauncher.Wait()

			if got, want := deps.Backend.givenWorkspaceName, "default"; got != want {
				t.Errorf("wrong workspace name in URL request\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := deps.Backend.givenTargetObject, test.wantTargetObject; got != want {
				t.Errorf("wrong target object in URL request\ngot:  %s\nwant: %s", got, want)
			}

			// One response in the mock browser launcher means one call to
			// the OpenURL method.
			if got, want := len(deps.BrowserLauncher.Responses), 1; got != want {
				t.Fatalf("wrong number of responses %d; want %d", got, want)
			}
			if got, want := deps.BrowserLauncher.Responses[0].Request.URL.String(), deps.TestServerURL; got != want {
				t.Errorf("wrong URL\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

type backendForTestWeb struct {
	// This is mostly just the inmem backend, but with the web URL provider
	// API implemented too.
	*inmem.Backend

	retURL *url.URL

	givenWorkspaceName string
	givenTargetObject  webcommand.TargetObject
}

var _ webcommand.URLProvider = (*backendForTestWeb)(nil)

func (b *backendForTestWeb) Operation(context.Context, *backend.Operation) (*backend.RunningOperation, error) {
	panic("not implemented")
}

func (b *backendForTestWeb) WebURLForObject(ctx context.Context, workspaceName string, targetObject webcommand.TargetObject) (*url.URL, tfdiags.Diagnostics) {
	b.givenWorkspaceName = workspaceName
	b.givenTargetObject = targetObject
	return b.retURL, nil
}
