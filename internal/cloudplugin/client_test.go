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
		err := client.DownloadFile("/archives/terraform-cloudplugin/terraform-cloudplugin_0.1.0_SHA256SUMS", &buffer)
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

		if manifest.lastModified != testManifestLastModified {
			t.Errorf("expected lastModified %q, got %q", manifest.lastModified, testManifestLastModified)
		}

		if expected := "0.1.0"; manifest.ProductVersion != expected {
			t.Errorf("expected ProductVersion %q, got %q", expected, manifest.ProductVersion)
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
