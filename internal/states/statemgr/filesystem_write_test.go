// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/zclconf/go-cty/cty"
)

var filesystemWriteMethods = map[string]func(*Filesystem, *statefile.File) error{
	"WriteState": func(s *Filesystem, f *statefile.File) error {
		return s.WriteState(f.State)
	},
	"WriteStateForMigration": func(s *Filesystem, f *statefile.File) error {
		return s.WriteStateForMigration(f, true)
	},
}

func TestFilesystem_unlockedWriteReleasesFile(t *testing.T) {
	for name, write := range filesystemWriteMethods {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "terraform.tfstate")
			s := NewFilesystem(path)
			for i := uint64(1); i <= 3; i++ {
				want := statefile.New(TestFullInitialState(), "test-lineage", i)
				if err := write(s, want); err != nil {
					t.Fatal(err)
				}
				got := readFilesystemTestState(t, path)
				if !got.State.Equal(want.State) {
					t.Fatal("written state does not match")
				}
				if name == "WriteStateForMigration" && (got.Lineage != want.Lineage || got.Serial != want.Serial) {
					t.Fatalf("migration metadata changed: got %q/%d, want %q/%d", got.Lineage, got.Serial, want.Lineage, want.Serial)
				}

				// Windows requires the handle to be closed before removal. On
				// Unix, the next write must recreate this path rather than write
				// to the old, unlinked file through a retained handle.
				if err := os.Remove(path); err != nil {
					t.Fatalf("state file remains open after writing: %s", err)
				}
			}
		})
	}
}

func TestFilesystem_unlockedWriteBackupError(t *testing.T) {
	for name, write := range filesystemWriteMethods {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.tfstate")
			output := filepath.Join(dir, "output.tfstate")
			backup := filepath.Join(dir, "backup.tfstate")
			original := statefile.New(TestFullInitialState(), "test-lineage", 1)
			writeFilesystemTestState(t, input, original)
			writeFilesystemTestState(t, output, original)
			s := NewFilesystemBetweenPaths(input, output)
			s.SetBackupPath(backup)
			if err := os.Mkdir(backup, 0700); err != nil {
				t.Fatal(err)
			}

			updated := original.DeepCopy()
			updated.Serial++
			updated.State.SetOutputValue(addrs.OutputValue{Name: "new"}.Absolute(addrs.RootModuleInstance), cty.StringVal("updated"), false)
			if err := write(s, updated); err == nil {
				t.Fatal("expected the write to fail")
			}
			got := readFilesystemTestState(t, output)
			if !got.State.Equal(original.State) {
				t.Fatal("failed backup overwrote the original state")
			}
			if err := os.Remove(backup); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(output); err != nil {
				t.Fatalf("failed write retained the output file: %s", err)
			}
			if err := write(s, updated); err != nil {
				t.Fatalf("failed to retry writing state: %s", err)
			}
			if got := readFilesystemTestState(t, output); !got.State.Equal(updated.State) {
				t.Fatal("retried write did not persist the new state")
			}
		})
	}
}

func TestFilesystem_unlockedWriteBackup(t *testing.T) {
	for _, initial := range []string{"missing", "empty", "existing"} {
		t.Run(initial, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "terraform.tfstate")
			backup := path + ".backup"
			original := statefile.New(TestFullInitialState(), "test-lineage", 3)
			switch initial {
			case "empty":
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "existing":
				writeFilesystemTestState(t, path, original)
			}

			s := NewFilesystem(path)
			s.SetBackupPath(backup)
			// An unchanged write should not create a backup. Subsequent writes
			// must still back up only the snapshot that predated this writer.
			if err := s.WriteState(original.State); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(backup); !os.IsNotExist(err) {
				t.Fatalf("unexpected backup after the initial write: %v", err)
			}
			for i := int64(1); i <= 2; i++ {
				updated := original.State.DeepCopy()
				updated.SetOutputValue(addrs.OutputValue{Name: "iteration"}.Absolute(addrs.RootModuleInstance), cty.NumberIntVal(i), false)
				if err := s.WriteState(updated); err != nil {
					t.Fatal(err)
				}
				if got := readFilesystemTestState(t, path); !got.State.Equal(updated) {
					t.Fatal("updated state does not match")
				}
				if initial == "existing" {
					got := readFilesystemTestState(t, backup)
					if !got.State.Equal(original.State) || got.Lineage != original.Lineage || got.Serial != original.Serial {
						t.Fatal("backup does not preserve the original snapshot and metadata")
					}
				} else if _, err := os.Stat(backup); !os.IsNotExist(err) {
					t.Fatalf("created a backup of a snapshot written by this manager: %v", err)
				}
			}
		})
	}
}

func TestFilesystem_writeRetainsLock(t *testing.T) {
	for name, write := range filesystemWriteMethods {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "terraform.tfstate")
			s := NewFilesystem(path)
			lockID, err := s.Lock(NewLockInfo())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := s.Unlock(lockID); err != nil {
					t.Error(err)
				}
			})
			want := statefile.New(TestFullInitialState(), "test-lineage", 1)
			if err := write(s, want); err != nil {
				t.Fatal(err)
			}
			if err := s.RefreshState(); err != nil {
				t.Fatalf("failed to refresh state while locked: %s", err)
			}
			if !s.State().Equal(want.State) {
				t.Fatal("refreshed state does not match")
			}
			output, err := exec.Command("go", "run", "testdata/lockstate.go", path).CombinedOutput()
			if err != nil {
				t.Fatalf("failed to run competing state locker: %s\n%s", err, output)
			}
			if !strings.Contains(string(output), "lock failed") {
				t.Fatalf("another process acquired the lock after writing: %s", output)
			}
		})
	}
}

func writeFilesystemTestState(t *testing.T, path string, state *statefile.File) {
	t.Helper()
	var buf bytes.Buffer
	if err := statefile.Write(state, &buf); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

func readFilesystemTestState(t *testing.T, path string) *statefile.File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state, err := statefile.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return state
}
