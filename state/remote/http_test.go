package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-cleanhttp"
)

func TestHTTPClient_impl(t *testing.T) {
	var _ Client = new(HTTPClient)
	var _ ClientLocker = new(HTTPClient)
}

func TestHTTPClient(t *testing.T) {
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()

	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	client := &HTTPClient{URL: url, Client: cleanhttp.DefaultClient()}
	testClient(t, client)

	a := &HTTPClient{
		URL:          url,
		LockURL:      url,
		LockMethod:   "LOCK",
		UnlockURL:    url,
		UnlockMethod: "UNLOCK",
		Client:       cleanhttp.DefaultClient(),
	}
	b := &HTTPClient{
		URL:          url,
		LockURL:      url,
		LockMethod:   "LOCK",
		UnlockURL:    url,
		UnlockMethod: "UNLOCK",
		Client:       cleanhttp.DefaultClient(),
	}
	TestRemoteLocks(t, a, b)
}

type testHTTPHandler struct {
	Data   []byte
	Locked bool
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Write(h.Data)
	case "POST":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(500)
		}

		h.Data = buf.Bytes()
	case "LOCK":
		if h.Locked {
			w.WriteHeader(409)
		} else {
			h.Locked = true
		}
	case "UNLOCK":
		h.Locked = false
	case "DELETE":
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}
