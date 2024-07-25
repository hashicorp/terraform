// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

var testManifest = `{
	"builds": [
		{
			"arch": "amd64",
			"os": "darwin",
			"url": "/archives/terraform-cloudplugin_0.1.0_darwin_amd64.zip"
		}
	],
	"is_prerelease": true,
	"license_class": "ent",
	"name": "terraform-cloudplugin",
	"status": {
		"state": "supported",
		"timestamp_updated": "2023-07-31T15:18:20.243Z"
	},
	"timestamp_created": "2023-07-31T15:18:20.243Z",
	"timestamp_updated": "2023-07-31T15:18:20.243Z",
	"url_changelog": "https://github.com/hashicorp/terraform-cloudplugin/blob/main/CHANGELOG.md",
	"url_license": "https://github.com/hashicorp/terraform-cloudplugin/blob/main/LICENSE",
	"url_project_website": "https://www.terraform.io/",
	"url_shasums": "/archives/terraform-cloudplugin_0.1.0_SHA256SUMS",
	"url_shasums_signatures": [
		"/archives/terraform-cloudplugin_0.1.0_SHA256SUMS.sig",
		"/archives/terraform-cloudplugin_0.1.0_SHA256SUMS.72D7468F.sig"
	],
	"url_source_repository": "https://github.com/hashicorp/terraform-cloudplugin",
	"version": "0.1.0"
}`

var (
	// This is the same as timestamp_updated above
	testManifestLastModified, _ = time.Parse(time.RFC3339, "2023-07-31T15:18:20Z")
)

type testHTTPHandler struct {
}

func (h *testHTTPHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	}

	switch r.URL.Path {
	case "/api/cloudplugin/v1/manifest.json":
		ifModifiedSince, _ := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
		w.Header().Set("Last-Modified", testManifestLastModified.Format(http.TimeFormat))

		if ifModifiedSince.Equal(testManifestLastModified) || testManifestLastModified.Before(ifModifiedSince) {
			w.WriteHeader(http.StatusNotModified)
		} else {
			w.Write([]byte(testManifest))
		}
	default:
		fileToSend, err := os.Open(fmt.Sprintf("testdata/%s", r.URL.Path))
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
