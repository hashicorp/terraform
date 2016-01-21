package remote

import (
	"bytes"
	"crypto/md5"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/terraform"
)

func TestAtlasClient_impl(t *testing.T) {
	var _ Client = new(AtlasClient)
}

func TestAtlasClient(t *testing.T) {
	acctest.RemoteTestPrecheck(t)

	token := os.Getenv("ATLAS_TOKEN")
	if token == "" {
		t.Skipf("skipping, ATLAS_TOKEN must be set")
	}

	client, err := atlasFactory(map[string]string{
		"access_token": token,
		"name":         "hashicorp/test-remote-state",
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}

func TestAtlasClient_ReportedConflictEqualStates(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateModuleOrderChange)
	srv := fakeAtlas.Server()
	defer srv.Close()
	client, err := atlasFactory(map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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

func TestAtlasClient_NoConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)
	srv := fakeAtlas.Server()
	defer srv.Close()
	client, err := atlasFactory(map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

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

func TestAtlasClient_LegitimateConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)
	srv := fakeAtlas.Server()
	defer srv.Close()
	client, err := atlasFactory(map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := terraform.ReadState(bytes.NewReader(testStateSimple))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Changing the state but not the serial. Should generate a conflict.
	state.RootModule().Outputs["drift"] = "happens"

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := client.Put(stateJson.Bytes()); err == nil {
		t.Fatal("Expected error from state conflict, got none.")
	}
}

func TestAtlasClient_UnresolvableConflict(t *testing.T) {
	fakeAtlas := newFakeAtlas(t, testStateSimple)

	// Something unexpected causes Atlas to conflict in a way that we can't fix.
	fakeAtlas.AlwaysConflict(true)

	srv := fakeAtlas.Server()
	defer srv.Close()
	client, err := atlasFactory(map[string]string{
		"access_token": "sometoken",
		"name":         "someuser/some-test-remote-state",
		"address":      srv.URL,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := terraform.ReadState(bytes.NewReader(testStateSimple))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var stateJson bytes.Buffer
	if err := terraform.WriteState(state, &stateJson); err != nil {
		t.Fatalf("err: %s", err)
	}
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		if err := client.Put(stateJson.Bytes()); err == nil {
			t.Fatal("Expected error from state conflict, got none.")
		}
	}()

	select {
	case <-doneCh:
		// OK
	case <-time.After(50 * time.Millisecond):
		t.Fatalf("Timed out after 50ms, probably because retrying infinitely.")
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
	currentState, err := terraform.ReadState(bytes.NewReader(f.state))
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
	switch req.Method {
	case "GET":
		// Respond with the current stored state.
		resp.Header().Set("Content-Type", "application/json")
		resp.Write(f.state)
	case "PUT":
		var buf bytes.Buffer
		buf.ReadFrom(req.Body)
		sum := md5.Sum(buf.Bytes())
		state, err := terraform.ReadState(&buf)
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
    "version": 1,
    "serial": 1,
    "modules": [
        {
            "path": [
                "root",
                "child2",
                "grandchild"
            ],
            "outputs": {
                "foo": "bar2"
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
                "foo": "bar1"
            },
            "resources": null
        }
    ]
}
`)

var testStateSimple = []byte(
	`{
    "version": 1,
    "serial": 1,
    "modules": [
        {
            "path": [
                "root"
            ],
            "outputs": {
                "foo": "bar"
            },
            "resources": null
        }
    ]
}
`)
