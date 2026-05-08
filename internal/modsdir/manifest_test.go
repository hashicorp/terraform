// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package modsdir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestManifestWriteSnapshotToDirReleasesFileHandleWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific file handle behavior check")
	}

	dir := t.TempDir()
	manifest := Manifest{
		"": {
			Key: "root",
			Dir: dir,
		},
	}

	if err := manifest.WriteSnapshotToDir(dir); err != nil {
		t.Fatalf("failed to write manifest snapshot: %s", err)
	}

	snapshotPath := filepath.Join(dir, ManifestSnapshotFilename)
	if err := os.Remove(snapshotPath); err != nil {
		t.Fatalf("failed to remove snapshot file; file may still be open: %s", err)
	}
}

func TestManifestWriteSnapshotToDirDoesNotLeakFileDescriptors(t *testing.T) {
	if _, err := os.Stat("/proc/self/fd"); err != nil {
		t.Skip("/proc/self/fd not available")
	}

	dir := t.TempDir()
	manifest := Manifest{
		"": {
			Key: "root",
			Dir: dir,
		},
	}

	startFDCount, err := countOpenFDs()
	if err != nil {
		t.Fatalf("failed to read starting fd count: %s", err)
	}

	for i := 0; i < 256; i++ {
		if err := manifest.WriteSnapshotToDir(dir); err != nil {
			t.Fatalf("failed to write manifest snapshot at iteration %d: %s", i, err)
		}
	}

	endFDCount, err := countOpenFDs()
	if err != nil {
		t.Fatalf("failed to read ending fd count: %s", err)
	}

	const toleratedDrift = 32
	if leaked := endFDCount - startFDCount; leaked > toleratedDrift {
		t.Fatalf("likely leaked file descriptors; started with %d and ended with %d", startFDCount, endFDCount)
	}
}

func countOpenFDs() (int, error) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}
