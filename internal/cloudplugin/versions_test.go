package cloudplugin

import (
	"context"
	"net/url"
	"os"
	"testing"
)

func TestVersionManager_Resolve(t *testing.T) {
	publicKey, err := os.ReadFile("testdata/sample.public.key")
	if err != nil {
		t.Fatal(err)
	}

	server, err := newCloudPluginManifestHTTPTestServer(t)
	if err != nil {
		t.Fatalf("could not create test server: %s", err)
	}
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	serviceURL := serverURL.JoinPath("/api/cloudplugin/v1")

	tempDir := t.TempDir()
	manager, err := NewVersionManager(context.Background(), tempDir, serviceURL, "darwin", "amd64")
	if err != nil {
		t.Fatalf("expected no err, got: %s", err)
	}
	manager.signingKey = string(publicKey)
	manager.binaryName = "toucan.txt" // The file contained in the test archive

	version, err := manager.Resolve()
	if err != nil {
		t.Fatalf("expected no err, got %s", err)
	}

	if version == nil {
		t.Fatal("expected non-nil version")
	}

	if version.ResolvedFromCache {
		t.Error("expected non-cached version on first call to Resolve")
	}

	_, err = os.Stat(version.BinaryLocation)
	if err != nil {
		t.Fatalf("expected no error when getting binary location, got %q", err)
	}

	if version.ProductVersion != "0.1.0" { // from sample manifest
		t.Errorf("expected product version %q, got %q", "0.1.0", version.ProductVersion)
	}

	// Resolving a second time should return a cached version
	version, err = manager.Resolve()
	if err != nil {
		t.Fatalf("expected no err, got %s", err)
	}

	if version == nil {
		t.Fatal("expected non-nil version")
	}

	if !version.ResolvedFromCache {
		t.Error("expected cached version on second call to Resolve")
	}
}
