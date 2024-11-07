// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func newHTTPTestServerUnsupported(t *testing.T) (*httptest.Server, error) {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found"))
	})), nil
}

func TestCloudPluginClient_DownloadFile(t *testing.T) {
	server, err := newCloudPluginManifestHTTPTestServer(t)
	if err != nil {
		t.Fatalf("could not create test server: %s", err)
	}
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	serviceURL := serverURL.JoinPath("/api/cloudplugin/v1")

	client, err := NewCloudPluginClient(context.Background(), serviceURL)
	if err != nil {
		t.Fatalf("could not create test client: %s", err)
	}

	t.Run("200 response", func(t *testing.T) {
		buffer := bytes.Buffer{}
		err := client.DownloadFile("/archives/terraform-cloudplugin_0.1.0_SHA256SUMS", &buffer)
		if err != nil {
			t.Fatal("expected no error")
		}
		if buffer.Len() == 0 {
			t.Error("expected data, but got none")
		}
	})

	t.Run("404 response", func(t *testing.T) {
		err := client.DownloadFile("/archives/nope.zip", io.Discard)
		if !errors.Is(err, ErrCloudPluginNotFound) {
			t.Fatalf("expected error %q, got %q", ErrCloudPluginNotFound, err)
		}
	})
}

func TestCloudPluginClient_FetchManifest(t *testing.T) {
	server, err := newCloudPluginManifestHTTPTestServer(t)
	if err != nil {
		t.Fatalf("could not create test server: %s", err)
	}
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	serviceURL := serverURL.JoinPath("/api/cloudplugin/v1")

	client, err := NewCloudPluginClient(context.Background(), serviceURL)
	if err != nil {
		t.Fatalf("could not create test client: %s", err)
	}

	t.Run("200 response", func(t *testing.T) {
		manifest, err := client.FetchManifest(time.Time{})
		if err != nil {
			t.Fatal("expected no error")
		}

		if manifest == nil {
			t.Fatal("expected manifest")
		}

		if expected := "0.1.0"; manifest.Version != expected {
			t.Errorf("expected ProductVersion %q, got %q", expected, manifest.Version)
		}
	})

	t.Run("304 response", func(t *testing.T) {
		manifest, err := client.FetchManifest(testManifestLastModified)
		if err != nil {
			t.Fatal("expected no error")
		}

		if manifest != nil {
			t.Fatalf("expected nil manifest, got %#v", manifest)
		}
	})
}

func TestCloudPluginClient_NotSupportedByTerraformCloud(t *testing.T) {
	server, err := newHTTPTestServerUnsupported(t)
	if err != nil {
		t.Fatalf("could not create test server: %s", err)
	}
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	serviceURL := serverURL.JoinPath("/api/cloudplugin/v1")

	client, err := NewCloudPluginClient(context.Background(), serviceURL)
	if err != nil {
		t.Fatalf("could not create test client: %s", err)
	}

	_, err = client.FetchManifest(time.Time{})
	if !errors.Is(err, ErrCloudPluginNotSupported) {
		t.Errorf("Expected ErrCloudPluginNotSupported, got %v", err)
	}
}

func TestRelease_PrimarySHASumsSignatureURL(t *testing.T) {
	example := Release{
		URLSHASumsSignatures: []string{
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.sig",
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS/72D7468F.sig", // Not quite right
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.72D7468F.sig",
		},
	}

	url, err := example.PrimarySHASumsSignatureURL()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if url != example.URLSHASumsSignatures[2] {
		t.Errorf("Expected URL %q, but got %q", example.URLSHASumsSignatures[2], url)
	}
}

func TestRelease_PrimarySHASumsSignatureURL_lowercase_should_match(t *testing.T) {
	example := Release{
		URLSHASumsSignatures: []string{
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.sig",
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.72d7468f.sig",
		},
	}

	url, err := example.PrimarySHASumsSignatureURL()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	// Not expected but technically fine since these are hex values
	if url != example.URLSHASumsSignatures[1] {
		t.Errorf("Expected URL %q, but got %q", example.URLSHASumsSignatures[1], url)
	}
}

func TestRelease_PrimarySHASumsSignatureURL_no_known_keys(t *testing.T) {
	example := Release{
		URLSHASumsSignatures: []string{
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.sig",
			"https://releases.hashicorp.com/terraform-cloudplugin/0.1.0-prototype/terraform-cloudplugin_0.1.0-prototype_SHA256SUMS.ABCDEF012.sig",
		},
	}

	url, err := example.PrimarySHASumsSignatureURL()
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	// Returns key with no ID
	if url != example.URLSHASumsSignatures[0] {
		t.Errorf("Expected URL %q, but got %q", example.URLSHASumsSignatures[0], url)
	}
}
