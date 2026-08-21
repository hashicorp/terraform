//go:build darwin || linux

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"golang.org/x/sys/unix"
)

// TestBinaryManager_latestManifestClosesCacheFile is a regression test for
// https://github.com/hashicorp/terraform/issues/38302: writing the plugin
// manifest cache previously leaked the file descriptor created by os.Create.
func TestBinaryManager_latestManifestClosesCacheFile(t *testing.T) {
	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)

	server, err := newCloudPluginManifestHTTPTestServer(t)
	if err != nil {
		t.Fatalf("could not create test server: %s", err)
	}
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	serviceURL := serverURL.JoinPath("/api/cloudplugin/v1")

	// Use a single manager so that repeated iterations share the same HTTP
	// client and connection pool; this test is only interested in leaked file
	// descriptors from writing the manifest cache.
	manager, err := NewCloudBinaryManager(context.Background(), t.TempDir(), "", serviceURL, "darwin", "amd64")
	if err != nil {
		t.Fatalf("expected no err, got: %s", err)
	}

	manifestCacheLocation := filepath.Join(manager.pluginDataDir, manager.host.String(), "manifest.json")

	// Do an initial fetch to warm up the HTTP connection pool, then take the
	// baseline file descriptor count.
	if _, err := manager.latestManifest(context.Background()); err != nil {
		t.Fatalf("fetching manifest on warmup iteration: %s", err)
	}
	before := countOpenPluginsharedFDs(t)

	const iterations = 16
	for i := range iterations {
		// Remove the cache file so that latestManifest takes the path that
		// writes a new manifest cache file on every iteration.
		if err := os.RemoveAll(manifestCacheLocation); err != nil {
			t.Fatalf("removing manifest cache on iteration %d: %s", i, err)
		}

		if _, err := manager.latestManifest(context.Background()); err != nil {
			t.Fatalf("fetching manifest on iteration %d: %s", i, err)
		}

		if _, err := os.Stat(manifestCacheLocation); err != nil {
			t.Fatalf("expected manifest cache %q to have been written on iteration %d: %s", manifestCacheLocation, i, err)
		}
	}

	after := countOpenPluginsharedFDs(t)
	if leaked := after - before; leaked > 2 {
		t.Fatalf("expected latestManifest to close its file descriptor, but open descriptor count increased by %d", leaked)
	}
}

func countOpenPluginsharedFDs(t *testing.T) int {
	t.Helper()

	var limit unix.Rlimit
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit); err != nil {
		t.Fatalf("reading RLIMIT_NOFILE: %s", err)
	}

	maxFD := int(limit.Cur)
	if maxFD > 4096 {
		maxFD = 4096
	}

	openDescriptors := 0
	for fd := range maxFD {
		if _, err := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0); err == nil {
			openDescriptors++
		} else if err != unix.EBADF {
			t.Fatalf("checking file descriptor %d: %s", fd, err)
		}
	}

	return openDescriptors
}
