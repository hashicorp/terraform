package statemgr

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"testing"

	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/states/statefile"
)

func TestFilesystem(t *testing.T) {
	ls := testFilesystem(t)
	defer os.Remove(ls.readPath)
	TestFull(t, ls)
}

func TestFilesystemRace(t *testing.T) {
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
	s := testFilesystem(t)
	defer os.Remove(s.readPath)

	// lock first
	info := NewLockInfo()
	info.Operation = "test"
	lockID, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", "testdata/lockstate.go", s.path).CombinedOutput()
	if err != nil {
		t.Fatal("unexpected lock failure", err, string(out))
	}

	if string(out) != "lock failed" {
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

func TestFilesystem_nonExist(t *testing.T) {
	ls := NewFilesystem("ishouldntexist")
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if state := ls.State(); state != nil {
		t.Fatalf("bad: %#v", state)
	}
}

func TestFilesystem_impl(t *testing.T) {
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
