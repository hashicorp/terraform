package state

import (
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestLocalState(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	TestState(t, ls)
}

func TestLocalStateRace(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)

	current := TestStateInitial()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ls.WriteState(current)
		}()
	}
}

func TestLocalStateLocks(t *testing.T) {
	s := testLocalState(t)
	defer os.Remove(s.Path)

	// lock first
	info := NewLockInfo()
	info.Operation = "test"
	lockID, err := s.Lock(info)
	if err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", "testdata/lockstate.go", s.Path).CombinedOutput()
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
func TestLocalState_writeWhileLocked(t *testing.T) {
	s := testLocalState(t)
	defer os.Remove(s.Path)

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

	if err := s.WriteState(TestStateInitial()); err != nil {
		t.Fatal(err)
	}
}

func TestLocalState_pathOut(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	ls := testLocalState(t)
	ls.PathOut = f.Name()
	defer os.Remove(ls.Path)

	TestState(t, ls)
}

func TestLocalState_nonExist(t *testing.T) {
	ls := &LocalState{Path: "ishouldntexist"}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if state := ls.State(); state != nil {
		t.Fatalf("bad: %#v", state)
	}
}

func TestLocalState_impl(t *testing.T) {
	var _ StateReader = new(LocalState)
	var _ StateWriter = new(LocalState)
	var _ StatePersister = new(LocalState)
	var _ StateRefresher = new(LocalState)
}

func testLocalState(t *testing.T) *LocalState {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = terraform.WriteState(TestStateInitial(), f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	ls := &LocalState{Path: f.Name()}
	if err := ls.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	return ls
}

// Make sure we can refresh while the state is locked
func TestLocalState_refreshWhileLocked(t *testing.T) {
	f, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	err = terraform.WriteState(TestStateInitial(), f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s := &LocalState{Path: f.Name()}
	defer os.Remove(s.Path)

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
	if readState == nil || readState.Lineage == "" {
		t.Fatal("missing state")
	}
}
