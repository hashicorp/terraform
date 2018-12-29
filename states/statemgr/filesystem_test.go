package statemgr

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-test/deep"
	version "github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestFilesystem(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	ls := testFilesystem(t)
	defer os.Remove(ls.readPath)
	TestFull(t, ls)
}

func TestFilesystemRace(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	ls := testFilesystem(t)
	defer os.Remove(ls.readPath)

	current := TestFullInitialState()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ls.WriteState(current)
		}()
	}
}

func TestFilesystemLocks(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	s := testFilesystem(t)
	defer os.Remove(s.readPath)

	// lock first
	info := NewLockInfo()
	info.Operation = "test"
	lockID, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", "-mod=vendor", "testdata/lockstate.go", s.path).CombinedOutput()
	if err != nil {
		t.Fatal("unexpected lock failure", err, string(out))
	}

	if !strings.Contains(string(out), "lock failed") {
		t.Fatal("expected 'locked failed', got", string(out))
	}

	// check our lock info
	lockInfo, err := s.lockInfo()
	if err != nil {
		t.Fatal(err)
	}

	if lockInfo.Operation != "test" {
		t.Fatalf("invalid lock info %#v\n", lockInfo)
	}

	// a noop, since we unlock on exit
	if err := s.Unlock(lockID); err != nil {
		t.Fatal(err)
	}

	// local locks can re-lock
	lockID, err = s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Unlock(lockID); err != nil {
		t.Fatal(err)
	}

	// we should not be able to unlock the same lock twice
	if err := s.Unlock(lockID); err == nil {
		t.Fatal("unlocking an unlocked state should fail")
	}

	// make sure lock info is gone
	lockInfoPath := s.lockInfoPath()
	if _, err := os.Stat(lockInfoPath); !os.IsNotExist(err) {
		t.Fatal("lock info not removed")
	}
}

// Verify that we can write to the state file, as Windows' mandatory locking
// will prevent writing to a handle different than the one that hold the lock.
func TestFilesystem_writeWhileLocked(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	s := testFilesystem(t)
	defer os.Remove(s.readPath)

	// lock first
	info := NewLockInfo()
	info.Operation = "test"
	lockID, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Unlock(lockID); err != nil {
			t.Fatal(err)
		}
	}()

	if err := s.WriteState(TestFullInitialState()); err != nil {
		t.Fatal(err)
	}
}

func TestFilesystem_pathOut(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testFilesystem(t)
	ls.path = f.Name()
	defer os.Remove(ls.path)

	TestFull(t, ls)
}

func TestFilesystem_backup(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testFilesystem(t)
	backupPath := f.Name()
	ls.SetBackupPath(backupPath)

	TestFull(t, ls)

	// The backup functionality should've saved a copy of the original state
	// prior to all of the modifications that TestFull does.
	bfh, err := os.Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	bf, err := statefile.Read(bfh)
	if err != nil {
		t.Fatal(err)
	}
	origState := TestFullInitialState()
	if !bf.State.Equal(origState) {
		for _, problem := range deep.Equal(origState, bf.State) {
			t.Error(problem)
		}
	}
}

// This test verifies a particularly tricky behavior where the input file
// is overridden and backups are enabled at the same time. This combination
// requires special care because we must ensure that when we create a backup
// it is of the original contents of the output file (which we're overwriting),
// not the contents of the input file (which is left unchanged).
func TestFilesystem_backupAndReadPath(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()

	workDir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %s", err)
	}
	defer os.RemoveAll(workDir)

	markerOutput := addrs.OutputValue{Name: "foo"}.Absolute(addrs.RootModuleInstance)

	outState := states.BuildState(func(ss *states.SyncState) {
		ss.SetOutputValue(
			markerOutput,
			cty.StringVal("from-output-state"),
			false, // not sensitive
		)
	})
	outFile, err := os.Create(filepath.Join(workDir, "output.tfstate"))
	if err != nil {
		t.Fatalf("failed to create temporary outFile %s", err)
	}
	defer outFile.Close()
	err = statefile.Write(&statefile.File{
		Lineage:          "-",
		Serial:           0,
		TerraformVersion: version.Must(version.NewVersion("1.2.3")),
		State:            outState,
	}, outFile)
	if err != nil {
		t.Fatalf("failed to write initial outfile state to %s: %s", outFile.Name(), err)
	}

	inState := states.BuildState(func(ss *states.SyncState) {
		ss.SetOutputValue(
			markerOutput,
			cty.StringVal("from-input-state"),
			false, // not sensitive
		)
	})
	inFile, err := os.Create(filepath.Join(workDir, "input.tfstate"))
	if err != nil {
		t.Fatalf("failed to create temporary inFile %s", err)
	}
	defer inFile.Close()
	err = statefile.Write(&statefile.File{
		Lineage:          "-",
		Serial:           0,
		TerraformVersion: version.Must(version.NewVersion("1.2.3")),
		State:            inState,
	}, inFile)
	if err != nil {
		t.Fatalf("failed to write initial infile state to %s: %s", inFile.Name(), err)
	}

	backupPath := outFile.Name() + ".backup"

	ls := NewFilesystemBetweenPaths(inFile.Name(), outFile.Name())
	ls.SetBackupPath(backupPath)

	newState := states.BuildState(func(ss *states.SyncState) {
		ss.SetOutputValue(
			markerOutput,
			cty.StringVal("from-new-state"),
			false, // not sensitive
		)
	})
	err = ls.WriteState(newState)
	if err != nil {
		t.Fatalf("failed to write new state: %s", err)
	}

	// The backup functionality should've saved a copy of the original contents
	// of the _output_ file, even though the first snapshot was read from
	// the _input_ file.
	t.Run("backup file", func(t *testing.T) {
		bfh, err := os.Open(backupPath)
		if err != nil {
			t.Fatal(err)
		}
		bf, err := statefile.Read(bfh)
		if err != nil {
			t.Fatal(err)
		}
		os := bf.State.OutputValue(markerOutput)
		if got, want := os.Value, cty.StringVal("from-output-state"); !want.RawEquals(got) {
			t.Errorf("wrong marker value in backup state file\ngot:  %#v\nwant: %#v", got, want)
		}
	})
	t.Run("output file", func(t *testing.T) {
		ofh, err := os.Open(outFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		of, err := statefile.Read(ofh)
		if err != nil {
			t.Fatal(err)
		}
		os := of.State.OutputValue(markerOutput)
		if got, want := os.Value, cty.StringVal("from-new-state"); !want.RawEquals(got) {
			t.Errorf("wrong marker value in backup state file\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}

func TestFilesystem_nonExist(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	ls := NewFilesystem("ishouldntexist")
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if state := ls.State(); state != nil {
		t.Fatalf("bad: %#v", state)
	}
}

func TestFilesystem_impl(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	var _ Reader = new(Filesystem)
	var _ Writer = new(Filesystem)
	var _ Persister = new(Filesystem)
	var _ Refresher = new(Filesystem)
	var _ Locker = new(Filesystem)
}

func testFilesystem(t *testing.T) *Filesystem {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("failed to create temporary file %s", err)
	}
	t.Logf("temporary state file at %s", f.Name())

	err = statefile.Write(&statefile.File{
		Lineage:          "test-lineage",
		Serial:           0,
		TerraformVersion: version.Must(version.NewVersion("1.2.3")),
		State:            TestFullInitialState(),
	}, f)
	if err != nil {
		t.Fatalf("failed to write initial state to %s: %s", f.Name(), err)
	}
	f.Close()

	ls := NewFilesystem(f.Name())
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("initial refresh failed: %s", err)
	}

	return ls
}

// Make sure we can refresh while the state is locked
func TestFilesystem_refreshWhileLocked(t *testing.T) {
	defer testOverrideVersion(t, "1.2.3")()
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = statefile.Write(&statefile.File{
		Lineage:          "test-lineage",
		Serial:           0,
		TerraformVersion: version.Must(version.NewVersion("1.2.3")),
		State:            TestFullInitialState(),
	}, f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()

	s := NewFilesystem(f.Name())
	defer os.Remove(s.path)

	// lock first
	info := NewLockInfo()
	info.Operation = "test"
	lockID, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := s.Unlock(lockID); err != nil {
			t.Fatal(err)
		}
	}()

	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}

	readState := s.State()
	if readState == nil {
		t.Fatal("missing state")
	}
}

func testOverrideVersion(t *testing.T, v string) func() {
	oldVersionStr := tfversion.Version
	oldPrereleaseStr := tfversion.Prerelease
	oldSemVer := tfversion.SemVer

	var newPrereleaseStr string
	if dash := strings.Index(v, "-"); dash != -1 {
		newPrereleaseStr = v[dash+1:]
		v = v[:dash]
	}

	newSemVer, err := version.NewVersion(v)
	if err != nil {
		t.Errorf("invalid override version %q: %s", v, err)
	}
	newVersionStr := newSemVer.String()

	tfversion.Version = newVersionStr
	tfversion.Prerelease = newPrereleaseStr
	tfversion.SemVer = newSemVer

	return func() { // reset function
		tfversion.Version = oldVersionStr
		tfversion.Prerelease = oldPrereleaseStr
		tfversion.SemVer = oldSemVer
	}
}
