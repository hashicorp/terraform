// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package workdir

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDirForcedPluginDirs(t *testing.T) {
	tmpDir := t.TempDir()

	dir := NewDir(tmpDir)
	// We'll use the default convention of a data dir nested inside the
	// working directory, so we don't need to override anything on "dir".

	want := []string(nil)
	got, err := dir.ForcedPluginDirs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong initial settings\n%s", diff)
	}

	fakeDir1 := filepath.Join(tmpDir, "boop1")
	fakeDir2 := filepath.Join(tmpDir, "boop2")
	err = dir.SetForcedPluginDirs([]string{fakeDir1, fakeDir2})
	if err != nil {
		t.Fatal(err)
	}

	want = []string{fakeDir1, fakeDir2}
	got, err = dir.ForcedPluginDirs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong updated settings\n%s", diff)
	}

	err = dir.SetForcedPluginDirs(nil)
	if err != nil {
		t.Fatal(err)
	}

	want = nil
	got, err = dir.ForcedPluginDirs()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong final settings, after reverting back to defaults\n%s", diff)
	}
}
