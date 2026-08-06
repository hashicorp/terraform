//go:build darwin || linux

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package modsdir

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"

	"golang.org/x/sys/unix"
)

func TestManifestWriteSnapshotToDirClosesFile(t *testing.T) {
	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)

	manifest := Manifest{
		"root": {
			Key: "root",
			Dir: "modules/root",
		},
	}

	baseDir := t.TempDir()
	before := countOpenFileDescriptors(t)

	const iterations = 32
	for i := range iterations {
		dir := filepath.Join(baseDir, fmt.Sprintf("manifest-%d", i))
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("creating manifest dir %d: %s", i, err)
		}

		if err := manifest.WriteSnapshotToDir(dir); err != nil {
			t.Fatalf("writing manifest %d: %s", i, err)
		}
	}

	after := countOpenFileDescriptors(t)
	if leaked := after - before; leaked > 2 {
		t.Fatalf("expected WriteSnapshotToDir to close its file descriptor, but open descriptor count increased by %d", leaked)
	}
}

func countOpenFileDescriptors(t *testing.T) int {
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
