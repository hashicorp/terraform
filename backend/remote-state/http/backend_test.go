package http

import (
	"github.com/hashicorp/terraform/backend"

	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestStatePath construct the path based on named state
func TestStatePath(t *testing.T) {
	cases := []struct {
		name          string
		wantStatePath string
		wantLockPath  string
	}{
		{"default", "/default.tfstate", "/default.tflock"},
		{"test", "/test.tfstate", "/test.tflock"},
	}
	for _, c := range cases {
		b := new(Backend)
		if got := b.statePath(c.name); got != c.wantStatePath {
			t.Errorf("statePath(%q) = %q, want %q", c.name, got, c.wantStatePath)
		}

		if got := b.lockPath(c.name); got != c.wantLockPath {
			t.Errorf("lockPath(%q) = %q, want %q", c.name, got, c.wantLockPath)
		}
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {

	//start an http server to test
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()
	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	urls := fmt.Sprintf("%s", url)
	// Build config
	config := map[string]interface{}{
		"address": urls,
	}

	//backends
	b := backend.TestBackendConfig(t, New(), config).(*Backend)
	//Test if backend address matches the URL
	if b.address != urls {
		t.Fatal("Incorrect url was provided.")
	}
}

func TestBackendLocked(t *testing.T) {
	//start an http server to test
	handler := new(testHTTPHandler)
	ts := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer ts.Close()
	url, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	urls := fmt.Sprintf("%s", url)
	// Build config
	config := map[string]interface{}{
		"address": urls,
	}
	//backends
	b1 := backend.TestBackendConfig(t, New(), config).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), config).(*Backend)
	//test backends
	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

type testHTTPHandler struct {
	Data   map[string][]byte
	Locked bool
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if h.Data == nil {
		// initialize a map that will store all tfstate and tflock files.
		h.Data = make(map[string][]byte)
	}
	switch r.Method {
	case http.MethodGet:
		switch r.URL.Path {
		case "/foo.tfstate":
			w.Write(h.Data["/foo.tfstate"])
		case "/foo.tflock":
			w.Write(h.Data["/foo.tflock"])
		case "/bar.tfstate":
			w.Write(h.Data["/bar.tfstate"])
		case "/bar.tflock":
			w.Write(h.Data["/bar.tflock"])
		case "/default.tfstate":
			w.Write(h.Data["/default.tfstate"])
		case "/default.tflock":
			w.Write(h.Data["/default.tflock"])
		case "/":
			// returns all keys(as states)
			// States() will
			var keys []string
			for key := range h.Data {
				keys = append(keys, key)
			}
			all := fmt.Sprint(strings.Join(keys, ","))
			alls := []byte(all)
			h.Data["/"] = alls
			w.Write(h.Data["/"])
		}

	case "PUT":
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.WriteHeader(201)

	case "POST":
		switch r.URL.Path {
		case "/foo.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			h.Data["/foo.tfstate"] = buf.Bytes()

		case "/bar.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			h.Data["/bar.tfstate"] = buf.Bytes()

		case "/default.tfstate":
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			h.Data["/default.tfstate"] = buf.Bytes()
		}

	case "LOCK":
		switch r.URL.Path {
		case "/default.tflock":
			if h.Locked {
				w.WriteHeader(http.StatusLocked)
				w.Write([]byte(h.Data["/default.tflock"]))
			} else {
				if _, ok := h.Data["/default.tflock"]; ok {
					w.WriteHeader(http.StatusConflict)
				} else {
					buf := new(bytes.Buffer)
					if _, err := io.Copy(buf, r.Body); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					h.Data["/default.tflock"] = buf.Bytes()
					h.Locked = true
				}
			}

		case "/foo.tflock":
			if h.Locked {
				w.WriteHeader(http.StatusLocked)
				w.Write([]byte(h.Data["/foo.tflock"]))
			} else {
				if _, ok := h.Data["/foo.tflock"]; ok {
					w.WriteHeader(http.StatusConflict)
				} else {
					buf := new(bytes.Buffer)
					if _, err := io.Copy(buf, r.Body); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					h.Data["/foo.tflock"] = buf.Bytes()
					h.Locked = true
				}
			}

		case "/bar.tflock":
			if h.Locked {
				w.WriteHeader(http.StatusLocked)
				w.Write([]byte(h.Data["/bar.tflock"]))
			} else {
				if _, ok := h.Data["/bar.tflock"]; ok {
					w.WriteHeader(http.StatusConflict)
				} else {
					buf := new(bytes.Buffer)
					if _, err := io.Copy(buf, r.Body); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
					}
					h.Data["/bar.tflock"] = buf.Bytes()
					h.Locked = true
				}
			}
		}

	case "UNLOCK":
		switch r.URL.Path {
		case "/default.tflock":
			h.Locked = false
			delete(h.Data, "/default.tflock")
		case "/foo.tflock":
			h.Locked = false
			delete(h.Data, "/foo.tflock")
		case "/bar.tflock":
			h.Locked = false
			delete(h.Data, "/bar.tflock")
		}

	case "DELETE":
		switch r.URL.Path {
		// Delete foo.tfstate
		case "/foo.tfstate":
			delete(h.Data, "/foo.tfstate")
			w.WriteHeader(200)
			// Delete bar.tfstate
		case "/bar.tfstate":
			delete(h.Data, "/bar.tfstate")
			w.WriteHeader(200)
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}

}
