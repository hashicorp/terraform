package http

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/states/remote"
)

func TestHTTPClient_impl(t *testing.T) {
	var _ remote.Client = new(httpClient)
	var _ remote.ClientLocker = new(httpClient)
}

func createTestServer() (*testHTTPHandler, *httptest.Server, *url.URL, error) {
	handler := new(testHTTPHandler)
	handler.Reset()
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))

	url, err := url.Parse(ts.URL)
	if err != nil {
		ts.Close()
		return nil, nil, nil, err
	}
	return handler, ts, url, nil
}

func TestHTTPClient(t *testing.T) {
	handler, ts, url, err := createTestServer()
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	defer ts.Close()

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
	handler.Reset()
	handler.webDav = true
	client = &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		Client:       retryablehttp.NewClient(),
	}
	remote.TestClient(t, client) // first time through: 201
	remote.TestClient(t, client) // second time, with identical data: 204

	// test an intermittent broken backend
	handler.Reset()
	handler.failNext = true
	remote.TestClient(t, client)

	// test a workspace backend
	handler.Reset()

	url2, _ := url.Parse("/state/workspace1")
	url3, _ := url.Parse("/workspace/list")

	// workspace list
	client = &httpClient{
		URL:                   url2,
		UpdateMethod:          "PUT",
		Client:                retryablehttp.NewClient(),
		WorkspaceListURL:      url3,
		WorkspaceDeleteMethod: "DELETE",
	}
	// disable retrys as to not hold up test
	client.Client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		return false, nil
	}

	// empty
	handler.Data["/workspace/list"] = "[]"

	res, err := client.WorkspaceList()
	if err != nil {
		t.Fatal("unexpected error from workspaceList")
	}
	if len(res) != 0 {
		t.Fatal("unexpected number of workspaces returned for empty array")
	}

	// multi
	handler.Data["/workspace/list"] = "[\"entry1\",\"entry2\"]"
	res, err = client.WorkspaceList()
	if err != nil {
		t.Fatal("unexpected error from workspaceList with multi")
	}
	if len(res) != 2 {
		t.Fatal("unexpected number of workspaces returned for populated array")
	}
	if res[0] != "entry1" || res[1] != "entry2" {
		t.Fatalf("workspace entries do not match expected values %+v", res)
	}

	// error code
	handler.fail = true
	res, err = client.WorkspaceList()
	if err == nil {
		t.Fatal("expected an error when http service returns an error code")
	}

	// bad json
	handler.Data["/workspace/list"] = "[\"entry1\",\"entry2]"
	res, err = client.WorkspaceList()
	if err == nil {
		t.Fatalf("expected an error when attempting to decode invalid json, got payload %#v", res)
	}

	// workspace delete
	handler.Reset()
	handler.Data["/state/workspace1"] = "{}"
	err = client.WorkspaceDelete(url2)
	if err != nil {
		t.Fatalf("unexpected error from workspace delete, %s", err)
	}
	if _, ok := handler.Data["/state/workspace1"]; ok {
		t.Fatal("workspace delete did not remove state")
	}

	// non exist
	handler.fail = true
	err = client.WorkspaceDelete(url2)
	if err == nil {
		t.Fatal("expected error with bad response code from workspace delete")
	}
}

type testHTTPHandler struct {
	failNext bool
	fail     bool
	webDav   bool
	Data     map[string]string
	Locked   map[string]bool
}

func (h *testHTTPHandler) Reset() {
	h.failNext = false
	h.fail = false
	h.webDav = false
	h.Data = nil
	h.Locked = nil
	h.Data = map[string]string{}
	h.Locked = map[string]bool{}
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.failNext || h.fail {
		w.WriteHeader(500)
		h.failNext = false
		return
	}
	path := r.URL.Path

	switch r.Method {
	case "GET":
		if d, ok := h.Data[path]; ok {
			w.Write([]byte(d))
		} else {
			w.WriteHeader(404)
		}
	case "PUT":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		bufAsString := string(buf.Bytes())

		// only difference from webdav function is 204 on match
		if d, ok := h.Data[path]; ok && h.webDav && bufAsString == d {
			w.WriteHeader(204)
		} else {
			w.WriteHeader(201)
		}

		h.Data[path] = bufAsString
	case "POST":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}
		h.Data[path] = string(buf.Bytes())
		w.WriteHeader(201)
	case "LOCK":
		if v, ok := h.Locked[path]; ok && v {
			w.WriteHeader(423)
		} else {
			h.Locked[path] = true
		}
	case "UNLOCK":
		delete(h.Locked, path)
	case "DELETE":
		delete(h.Data, path)
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}
