package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/states/remote"
)

func TestHTTPClient_impl(t *testing.T) {
	var _ remote.Client = new(httpClient)
	var _ remote.ClientLocker = new(httpClient)
}

func TestHTTPClient(t *testing.T) {
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()

	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}

	// Test basic get/update
	client := &httpClient{URL: url, Client: retryablehttp.NewClient()}
	remote.TestClient(t, client)

	// test just a single PUT
	p := &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		Client:       retryablehttp.NewClient(),
	}
	remote.TestClient(t, p)

	// Test locking and alternative UpdateMethod
	a := &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		LockURL:      url,
		LockMethod:   "LOCK",
		UnlockURL:    url,
		UnlockMethod: "UNLOCK",
		Client:       retryablehttp.NewClient(),
	}
	b := &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		LockURL:      url,
		LockMethod:   "LOCK",
		UnlockURL:    url,
		UnlockMethod: "UNLOCK",
		Client:       retryablehttp.NewClient(),
	}
	remote.TestRemoteLocks(t, a, b)

	// test a WebDAV-ish backend
	davhandler := new(testHTTPHandler)
	ts = httptest.NewServer(http.HandlerFunc(davhandler.HandleWebDAV))
	defer ts.Close()

	url, err = url.Parse(ts.URL)
	client = &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		Client:       retryablehttp.NewClient(),
	}
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}

	remote.TestClient(t, client) // first time through: 201
	remote.TestClient(t, client) // second time, with identical data: 204

	// test a broken backend
	brokenHandler := new(testBrokenHTTPHandler)
	brokenHandler.handler = new(testHTTPHandler)
	ts = httptest.NewServer(http.HandlerFunc(brokenHandler.Handle))
	defer ts.Close()

	url, err = url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	client = &httpClient{URL: url, Client: retryablehttp.NewClient()}
	remote.TestClient(t, client)

	// Test workspaces
	workspacehandler := new(testHTTPHandler)
	ts = httptest.NewServer(http.HandlerFunc(workspacehandler.HandleWorkspaces))
	defer ts.Close()

	url, err = url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	client = &httpClient{
		URL:              url,
		WorkspacesURL:    url,
		WorkspacesMethod: "OPTIONS",
		Client:           retryablehttp.NewClient(),
		workspace:        "test-workspace",
	}
	remote.TestClient(t, client)

	workspaces, err := client.Workspaces()
	if err != nil {
		t.Fatalf("Failed to get workspaces: %s", err)
	}
	expectedWorkspaces := []string{"test-workspace", "test-workspace2"}
	if !reflect.DeepEqual(workspaces, expectedWorkspaces) {
		t.Fatalf("Workspaces %s do not match expected workspaces %s", workspaces, expectedWorkspaces)
	}
}

type testHTTPHandler struct {
	Data   []byte
	Locked bool
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Should not set this query parameter on default workspace
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		w.Write(h.Data)
	case "PUT":
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(201)
		h.Data = buf.Bytes()
	case "POST":
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		h.Data = buf.Bytes()
	case "LOCK":
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		if h.Locked {
			w.WriteHeader(423)
		} else {
			h.Locked = true
		}
	case "UNLOCK":
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		h.Locked = false
	case "DELETE":
		_, found := r.URL.Query()["workspace"]
		if found {
			w.WriteHeader(500)
		}
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}

// mod_dav-ish behavior
func (h *testHTTPHandler) HandleWebDAV(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Write(h.Data)
	case "PUT":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		if reflect.DeepEqual(h.Data, buf.Bytes()) {
			h.Data = buf.Bytes()
			w.WriteHeader(204)
		} else {
			h.Data = buf.Bytes()
			w.WriteHeader(201)
		}
	case "DELETE":
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}

// Test workspaces
func (h *testHTTPHandler) HandleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		workspace := r.URL.Query().Get("workspace")
		if workspace != "test-workspace" {
			w.WriteHeader(500)
		}
		w.Write(h.Data)
	case "PUT":
		workspace := r.URL.Query().Get("workspace")
		if workspace != "test-workspace" {
			w.WriteHeader(500)
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(201)
		h.Data = buf.Bytes()
	case "POST":
		workspace := r.URL.Query().Get("workspace")
		if workspace != "test-workspace" {
			w.WriteHeader(500)
		}
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		h.Data = buf.Bytes()
	case "DELETE":
		workspace := r.URL.Query().Get("workspace")
		if workspace != "test-workspace" {
			w.WriteHeader(500)
		}
		h.Data = nil
		w.WriteHeader(200)
	case "OPTIONS":
		workspaces, err := json.Marshal([]string{"test-workspace", "test-workspace2"})
		if err != nil {
			w.WriteHeader(500)
		}
		w.WriteHeader(200)
		w.Write(workspaces)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}

type testBrokenHTTPHandler struct {
	lastRequestWasBroken bool
	handler              *testHTTPHandler
}

func (h *testBrokenHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.lastRequestWasBroken {
		h.lastRequestWasBroken = false
		h.handler.Handle(w, r)
	} else {
		h.lastRequestWasBroken = true
		w.WriteHeader(500)
	}
}
