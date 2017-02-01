package state

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestLocalState(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	TestState(t, ls)
}

func TestLocalStateLocks(t *testing.T) {
	s := testLocalState(t)
	defer os.Remove(s.Path)

	// lock first
	if err := s.Lock("test"); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", "testdata/lockstate.go", s.Path).CombinedOutput()

	if err != nil {
		t.Fatal("unexpected lock failure", err)
	}

	if string(out) != "lock failed" {
		t.Fatal("expected 'locked failed', got", string(out))
	}

	// check our lock info
	lockInfo, err := s.lockInfo()
	if err != nil {
		t.Fatal(err)
	}

	if lockInfo.Reason != "test" {
		t.Fatalf("invalid lock info %#v\n", lockInfo)
	}

	// a noop, since we unlock on exit
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}

	// local locks can re-lock
	if err := s.Lock("test"); err != nil {
		t.Fatal(err)
	}

	// Unlock should be repeatable
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}
	if err := s.Unlock(); err != nil {
		t.Fatal(err)
	}

	// make sure lock info is gone
	lockInfoPath := s.lockInfoPath()
	if _, err := os.Stat(lockInfoPath); !os.IsNotExist(err) {
		t.Fatal("lock info not removed")
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
