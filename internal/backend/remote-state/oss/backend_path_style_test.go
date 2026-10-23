// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package oss

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/internal/backend"
)

func TestBackendForcePathStyle(t *testing.T) {
	// Use only the explicit test configuration, regardless of local credentials,
	// assume-role settings, or proxy configuration.
	for _, name := range []string{
		"ALICLOUD_PROFILE", "ALIBABA_CLOUD_PROFILE",
		"ALICLOUD_ASSUME_ROLE_ARN", "ALIBABA_CLOUD_ROLE_ARN",
		"ALICLOUD_SECURITY_TOKEN", "ALIBABA_CLOUD_SECURITY_TOKEN",
		"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY",
		"http_proxy", "https_proxy", "all_proxy",
	} {
		t.Setenv(name, "")
	}
	previousConfig := providerConfig
	providerConfig = nil
	t.Cleanup(func() { providerConfig = previousConfig })

	const privateHost = "ep-example.oss.ap-southeast-5.privatelink.aliyuncs.com"
	const publicHost = "oss-ap-southeast-5.aliyuncs.com"
	const bucketName = "terraform-state"
	const objectKey = "path/mystate/version-1.tfstate"

	for _, tc := range []struct {
		name     string
		endpoint string
		style    *bool
		wantHost string
		wantPath string
	}{
		{"default", "https://" + publicHost, nil, bucketName + "." + publicHost, "/" + objectKey},
		{"disabled", "https://" + publicHost, boolPtr(false), bucketName + "." + publicHost, "/" + objectKey},
		{"enabled", "https://" + publicHost, boolPtr(true), publicHost, "/" + bucketName + "/" + objectKey},
		{"privatelink_default", "https://" + privateHost, nil, bucketName + "." + privateHost, "/" + objectKey},
		{"privatelink_disabled", "https://" + privateHost, boolPtr(false), bucketName + "." + privateHost, "/" + objectKey},
		{"privatelink_enabled", "https://" + privateHost, boolPtr(true), privateHost, "/" + bucketName + "/" + objectKey},
		{"privatelink_without_scheme", privateHost, boolPtr(true), privateHost, "/" + bucketName + "/" + objectKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]interface{}{
				"access_key": "test-access-key",
				"secret_key": "test-secret-key",
				"region":     "ap-southeast-5",
				"endpoint":   tc.endpoint,
				"bucket":     bucketName,
				"prefix":     "path/mystate",
				"key":        "version-1.tfstate",
			}
			if tc.style != nil {
				config["force_path_style"] = *tc.style
			}
			b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)
			bucket, err := b.ossClient.Bucket(bucketName)
			if err != nil {
				t.Fatal(err)
			}
			signedURL, err := bucket.SignURL(objectKey, oss.HTTPGet, 60)
			if err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(signedURL)
			if err != nil {
				t.Fatal(err)
			}
			checkURL := func(u *url.URL, scheme string) {
				t.Helper()
				if u.Scheme != scheme || u.Host != tc.wantHost || u.Path != tc.wantPath {
					t.Errorf("unexpected URL: got %s://%s%s, want %s://%s%s", u.Scheme, u.Host, u.Path, scheme, tc.wantHost, tc.wantPath)
				}
			}
			checkURL(u, "https")

			// Exercise actual state requests too, without DNS lookups or cloud access.
			const state = `{"version":4}`
			methods := make(chan string, 4)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				checkURL(req.URL, "http")
				select {
				case methods <- req.Method:
				default:
					t.Errorf("unexpected extra request: %s", req.Method)
				}
				if req.Method == http.MethodPut {
					body, err := io.ReadAll(req.Body)
					if err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					if string(body) != state {
						t.Errorf("unexpected state upload: %q", body)
					}
				}
				if req.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				io.WriteString(w, state)
			}))
			defer server.Close()
			// An HTTP proxy captures SDK requests to the configured hostname
			// without resolving it. HTTPS URL generation was checked above.
			t.Setenv("HTTP_PROXY", server.URL)
			t.Setenv("NO_PROXY", "")
			t.Setenv("no_proxy", "")
			config["endpoint"] = "http://" + strings.TrimPrefix(tc.endpoint, "https://")
			b = backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

			client, err := b.remoteClient(backend.DefaultStateName)
			if err != nil {
				t.Fatal(err)
			}
			if diags := client.Put([]byte(state)); diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			payload, diags := client.Get()
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			if payload == nil || !bytes.Equal(payload.Data, []byte(state)) {
				t.Fatalf("unexpected state payload: %#v", payload)
			}
			if diags := client.Delete(); diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			for _, want := range []string{"PUT", "HEAD", "GET", "DELETE"} {
				select {
				case got := <-methods:
					if got != want {
						t.Errorf("unexpected request: got %s, want %s", got, want)
					}
				default:
					t.Errorf("missing %s request", want)
				}
			}
		})
	}
}

func boolPtr(v bool) *bool { return &v }
