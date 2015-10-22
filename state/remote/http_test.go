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
}

type testHTTPHandler struct {
	Data []byte
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
	case "DELETE":
		h.Data = nil
		w.WriteHeader(200)
	default:
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("Unknown method: %s", r.Method)))
	}
}
