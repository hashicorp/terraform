package retryablehttp

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequest(t *testing.T) {
	// Fails on invalid request
	_, err := NewRequest("GET", "://foo", nil)
	if err == nil {
		t.Fatalf("should error")
	}

	// Works with no request body
	_, err = NewRequest("GET", "http://foo", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Works with request body
	body := bytes.NewReader([]byte("yo"))
	req, err := NewRequest("GET", "/", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Request allows typical HTTP request forming methods
	req.Header.Set("X-Test", "foo")
	if v, ok := req.Header["X-Test"]; !ok || len(v) != 1 || v[0] != "foo" {
		t.Fatalf("bad headers: %v", req.Header)
	}

	// Sets the Content-Length automatically for LenReaders
	if req.ContentLength != 2 {
		t.Fatalf("bad ContentLength: %d", req.ContentLength)
	}
}

func TestClient_Do(t *testing.T) {
	// Create a request
	body := bytes.NewReader([]byte("hello"))
	req, err := NewRequest("PUT", "http://127.0.0.1:28934/v1/foo", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req.Header.Set("foo", "bar")

	// Create the client. Use short retry windows.
	client := NewClient()
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 50 * time.Millisecond
	client.RetryMax = 50

	// Send the request
	var resp *http.Response
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		var err error
		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
	}()

	select {
	case <-doneCh:
		t.Fatalf("should retry on error")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to connection failure.
	}

	// Create the mock handler. First we return a 500-range response to ensure
	// that we power through and keep retrying in the face of recoverable
	// errors.
	code := int64(500)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request details
		if r.Method != "PUT" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/v1/foo" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}

		// Check the headers
		if v := r.Header.Get("foo"); v != "bar" {
			t.Fatalf("bad header: expect foo=bar, got foo=%v", v)
		}

		// Check the payload
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte("hello")
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(int(atomic.LoadInt64(&code)))
	})

	// Create a test server
	list, err := net.Listen("tcp", ":28934")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer list.Close()
	go http.Serve(list, handler)

	// Wait again
	select {
	case <-doneCh:
		t.Fatalf("should retry on 500-range")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to 500's.
	}

	// Start returning 200's
	atomic.StoreInt64(&code, 200)

	// Wait again
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		t.Fatalf("timed out")
	}

	if resp.StatusCode != 200 {
		t.Fatalf("exected 200, got: %d", resp.StatusCode)
	}
}

func TestClient_Do_fails(t *testing.T) {
	// Mock server which always responds 500.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	// Create the client. Use short retry windows so we fail faster.
	client := NewClient()
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 10 * time.Millisecond
	client.RetryMax = 2

	// Create the request
	req, err := NewRequest("POST", ts.URL, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Send the request.
	_, err = client.Do(req)
	if err == nil || !strings.Contains(err.Error(), "giving up") {
		t.Fatalf("expected giving up error, got: %#v", err)
	}
}

func TestClient_Get(t *testing.T) {
	// Mock server which always responds 500.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Get(ts.URL + "/foo/bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_Head(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Head(ts.URL + "/foo/bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_Post(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("bad content-type: %s", ct)
		}

		// Check the payload
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte(`{"hello":"world"}`)
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Post(
		ts.URL+"/foo/bar",
		"application/json",
		strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_PostForm(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("bad content-type: %s", ct)
		}

		// Check the payload
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte(`hello=world`)
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Create the form data.
	form, err := url.ParseQuery("hello=world")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Make the request.
	resp, err := NewClient().PostForm(ts.URL+"/foo/bar", form)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestBackoff(t *testing.T) {
	type tcase struct {
		min    time.Duration
		max    time.Duration
		i      int
		expect time.Duration
	}
	cases := []tcase{
		{
			time.Second,
			5 * time.Minute,
			0,
			time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			1,
			2 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			2,
			4 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			3,
			8 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			63,
			5 * time.Minute,
		},
		{
			time.Second,
			5 * time.Minute,
			128,
			5 * time.Minute,
		},
	}

	for _, tc := range cases {
		if v := backoff(tc.min, tc.max, tc.i); v != tc.expect {
			t.Fatalf("bad: %#v -> %s", tc, v)
		}
	}
}
