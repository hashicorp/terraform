package atlas

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/hashicorp/terraform/terraform"
)

func testStateClient(t *testing.T, c map[string]string) remote.Client {
	vals := make(map[string]cty.Value)
	for k, s := range c {
		vals[k] = cty.StringVal(s)
	}
	synthBody := configs.SynthBody("<test>", vals)

	b := backend.TestBackendConfig(t, New(), synthBody)
	raw, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s := raw.(*remote.State)
	return s.Client
}

func TestStateClient_impl(t *testing.T) {
	var _ remote.Client = new(stateClient)
}

func TestStateClient(t *testing.T) {
	acctest.RemoteTestPrecheck(t)

	token := os.Getenv("ATLAS_TOKEN")
	if token == "" {
		t.Skipf("skipping, ATLAS_TOKEN must be set")
	}

	client := testStateClient(t, map[string]string{
		"access_token": token,
		"name":         "hashicorp/test-remote-state",
	})

	remote.TestClient(t, client)
}

func TestStateClient_noRetryOnBadCerts(t *testing.T) {
	acctest.RemoteTestPrecheck(t)

	client := testStateClient(t, map[string]string{
		"access_token": "NOT_REQUIRED",
		"name":         "hashicorp/test-remote-state",
	})

	ac := client.(*stateClient)
	// trigger the StateClient to build the http client and assign HTTPClient
	httpClient, err := ac.http()
	if err != nil {
		t.Fatal(err)
	}

	// remove the CA certs from the client
	brokenCfg := &tls.Config{
		RootCAs: new(x509.CertPool),
	}
	httpClient.HTTPClient.Transport.(*http.Transport).TLSClientConfig = brokenCfg

	// Instrument CheckRetry to make sure we didn't retry
	retries := 0
	oldCheck := httpClient.CheckRetry
	httpClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if retries > 0 {
			t.Fatal("retried after certificate error")
		}
		retries++
		return oldCheck(ctx, resp, err)
	}

	_, err = client.Get()
	if err != nil {
		if err, ok := err.(*url.Error); ok {
			if _, ok := err.Err.(x509.UnknownAuthorityError); ok {
				return
			}
		}
	}

	t.Fatalf("expected x509.UnknownAuthorityError, got %v", err)
}

func TestStateClient_ReportedConflictEqualStates(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateModuleOrderChange)
	srv := fakeAtlas.Server()
	defer srv.Close()

	client := testStateClient(t, map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})

	state, err := terraform.ReadState(bytes.NewReader(testStateModuleOrderChange))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := client.Put(stateJson.Bytes()); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestStateClient_NoConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)
	srv := fakeAtlas.Server()
	defer srv.Close()

	client := testStateClient(t, map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})

	state, err := terraform.ReadState(bytes.NewReader(testStateSimple))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	fakeAtlas.NoConflictAllowed(true)

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := client.Put(stateJson.Bytes()); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestStateClient_LegitimateConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)
	srv := fakeAtlas.Server()
	defer srv.Close()

	client := testStateClient(t, map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})

	state, err := terraform.ReadState(bytes.NewReader(testStateSimple))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var buf bytes.Buffer
	terraform.WriteState(state, &buf)

	// Changing the state but not the serial. Should generate a conflict.
	state.RootModule().Outputs["drift"] = &terraform.OutputState{
		Type:      "string",
		Sensitive: false,
		Value:     "happens",
	}

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := client.Put(stateJson.Bytes()); err == nil {
		t.Fatal("Expected error from state conflict, got none.")
	}
}

func TestStateClient_UnresolvableConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)

	// Something unexpected causes Atlas to conflict in a way that we can't fix.
	fakeAtlas.AlwaysConflict(true)

	srv := fakeAtlas.Server()
	defer srv.Close()

	client := testStateClient(t, map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})

	state, err := terraform.ReadState(bytes.NewReader(testStateSimple))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if err := client.Put(stateJson.Bytes()); err == nil {
			errCh <- errors.New("expected error from state conflict, got none.")
			return
		}
	}()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("error from anonymous test goroutine: %s", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("Timed out after 500ms, probably because retrying infinitely.")
	}
}

// Stub Atlas HTTP API for a given state JSON string; does checksum-based
// conflict detection equivalent to Atlas's.
type fakeAtlas struct {
	state []byte
	t     *testing.T

	// Used to test that we only do the special conflict handling retry once.
	alwaysConflict bool

	// Used to fail the test immediately if a conflict happens.
	noConflictAllowed bool
}

func newFakeAtlas(t *testing.T, state []byte) *fakeAtlas {
	return &fakeAtlas{
		state: state,
		t:     t,
	}
}

func (f *fakeAtlas) Server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(f.handler))
}

func (f *fakeAtlas) CurrentState() *terraform.State {
	// we read the state manually here, because terraform may alter state
	// during read
	currentState := &terraform.State{}
	err := json.Unmarshal(f.state, currentState)
	if err != nil {
		f.t.Fatalf("err: %s", err)
	}
	return currentState
}

func (f *fakeAtlas) CurrentSerial() int64 {
	return f.CurrentState().Serial
}

func (f *fakeAtlas) CurrentSum() [md5.Size]byte {
	return md5.Sum(f.state)
}

func (f *fakeAtlas) AlwaysConflict(b bool) {
	f.alwaysConflict = b
}

func (f *fakeAtlas) NoConflictAllowed(b bool) {
	f.noConflictAllowed = b
}

func (f *fakeAtlas) handler(resp http.ResponseWriter, req *http.Request) {
	// access tokens should only be sent as a header
	if req.FormValue("access_token") != "" {
		http.Error(resp, "access_token in request params", http.StatusBadRequest)
		return
	}

	if req.Header.Get(atlasTokenHeader) == "" {
		http.Error(resp, "missing access token", http.StatusBadRequest)
		return
	}

	switch req.Method {
	case "GET":
		// Respond with the current stored state.
		resp.Header().Set("Content-Type", "application/json")
		resp.Write(f.state)
	case "PUT":
		var buf bytes.Buffer
		buf.ReadFrom(req.Body)
		sum := md5.Sum(buf.Bytes())

		// we read the state manually here, because terraform may alter state
		// during read
		state := &terraform.State{}
		err := json.Unmarshal(buf.Bytes(), state)
		if err != nil {
			f.t.Fatalf("err: %s", err)
		}

		conflict := f.CurrentSerial() == state.Serial && f.CurrentSum() != sum
		conflict = conflict || f.alwaysConflict
		if conflict {
			if f.noConflictAllowed {
				f.t.Fatal("Got conflict when NoConflictAllowed was set.")
			}
			http.Error(resp, "Conflict", 409)
		} else {
			f.state = buf.Bytes()
			resp.WriteHeader(200)
		}
	}
}

// This is a tfstate file with the module order changed, which is a structural
// but not a semantic difference. Terraform will sort these modules as it
// loads the state.
var testStateModuleOrderChange = []byte(
	`{
    "version": 3,
    "serial": 1,
    "modules": [
        {
            "path": [
                "root",
                "child2",
                "grandchild"
            ],
            "outputs": {
                "foo": {
                    "sensitive": false,
                    "type": "string",
                    "value": "bar"
                }
            },
            "resources": null
        },
        {
            "path": [
                "root",
                "child1",
                "grandchild"
            ],
            "outputs": {
                "foo": {
                    "sensitive": false,
                    "type": "string",
                    "value": "bar"
                }
            },
            "resources": null
        }
    ]
}
`)

var testStateSimple = []byte(
	`{
    "version": 3,
    "serial": 2,
    "lineage": "c00ad9ac-9b35-42fe-846e-b06f0ef877e9",
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {
                "foo": {
                    "sensitive": false,
                    "type": "string",
                    "value": "bar"
                }
            },
            "resources": {},
            "depends_on": []
        }
    ]
}
`)
