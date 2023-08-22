package cloudplugin

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"
)

var testManifest = `{
  "plugin_version": "0.1.0",
  "archives": {
    "darwin_amd64": {
      "url": "/archives/terraform-cloudplugin/terraform-cloudplugin_0.1.0_darwin_amd64.zip",
      "sha256sum": "22db2f0c70b50cff42afd4878fea9f6848a63f1b6532bd8b64b899f574acb35d"
    }
  },
  "sha256sums_url": "/archives/terraform-cloudplugin/terraform-cloudplugin_0.1.0_SHA256SUMS",
  "sha256sums_signature_url": "/archives/terraform-cloudplugin/terraform-cloudplugin_0.1.0_SHA256SUMS.sig"
}`

var (
	testManifestLastModified = time.Date(2023, time.August, 1, 0, 0, 0, 0, time.UTC)
)

type testHTTPHandler struct {
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	}

	switch r.URL.Path {
	case "/api/cloudplugin/v1/manifest":
		ifModifiedSince, _ := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
		w.Header().Set("Last-Modified", testManifestLastModified.Format(http.TimeFormat))

		if ifModifiedSince.Equal(testManifestLastModified) || testManifestLastModified.Before(ifModifiedSince) {
			w.WriteHeader(http.StatusNotModified)
		} else {
			w.Write([]byte(testManifest))
		}
	default:
		baseName := path.Base(r.URL.Path)
		fileToSend, err := os.Open(fmt.Sprintf("testdata/archives/%s", baseName))
		if err == nil {
			io.Copy(w, fileToSend)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	}
}

func newCloudPluginManifestHTTPTestServer(t *testing.T) (*httptest.Server, error) {
	t.Helper()

	handler := new(testHTTPHandler)
	return httptest.NewServer(http.HandlerFunc(handler.Handle)), nil
}
