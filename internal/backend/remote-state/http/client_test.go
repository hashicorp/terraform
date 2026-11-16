// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestHTTPClient_impl(t *testing.T) {
	var _ remote.Client = new(httpClient)
	var _ remote.ClientLocker = new(httpClient)
}

func TestHTTPClient(t *testing.T) {
	handler := new(TestHTTPBackend)
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
	davhandler := new(TestHTTPBackend)
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
	brokenHandler := new(TestBrokenHTTPBackend)
	brokenHandler.handler = new(TestHTTPBackend)
	ts = httptest.NewServer(http.HandlerFunc(brokenHandler.Handle))
	defer ts.Close()

	url, err = url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}
	client = &httpClient{URL: url, Client: retryablehttp.NewClient()}
	remote.TestClient(t, client)
}

// Make assertions about the data returned in lock errors
func TestHTTPClient_lockErrors(t *testing.T) {
	// Create a test HTTP backend that's already locked and contains
	// data about the current lock.
	testOperation := "test-setup-lock"
	testWho := "i-am-the-one-who-locks"
	handler := new(TestHTTPBackend)
	handler.Locked = true
	handler.LockInfo = &statemgr.LockInfo{
		Operation: testOperation,
		Who:       testWho,
	}
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()

	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("Parse: %s", err)
	}

	// Test locking when the test server is set up to already be locked.
	var locker statemgr.Locker = &httpClient{
		URL:          url,
		UpdateMethod: "PUT",
		LockURL:      url,
		LockMethod:   "LOCK",
		UnlockURL:    url,
		UnlockMethod: "UNLOCK",
		Client:       retryablehttp.NewClient(),
	}

	// Attempt to acquire a new lock with the data below
	info := statemgr.NewLockInfo()
	info.Operation = "can-i-get-a-lock?"
	info.Who = "client-that-wants-the-lock"
	_, err = locker.Lock(info)

	// Assert expected outcome: an error mentioning the pre-existing lock.
	if err == nil {
		t.Fatal("test client obtained lock while the server was locked by another client")
	}
	lockErr, ok := err.(*statemgr.LockError)
	if !ok {
		t.Errorf("expected a LockError, but was %t: %s", err, err)
	}
	if lockErr.Info.Operation != testOperation {
		t.Errorf("expected lock info operation %q, but was %q", testOperation, lockErr.Info.Operation)
	}
	if lockErr.Info.Who != testWho {
		t.Errorf("expected lock held by %q, but was %q", testWho, lockErr.Info.Who)
	}
}

type TestBrokenHTTPBackend struct {
	lastRequestWasBroken bool
	handler              *TestHTTPBackend
}

func (h *TestBrokenHTTPBackend) Handle(w http.ResponseWriter, r *http.Request) {
	if h.lastRequestWasBroken {
		h.lastRequestWasBroken = false
		h.handler.Handle(w, r)
	} else {
		h.lastRequestWasBroken = true
		w.WriteHeader(500)
	}
}
