package state

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestLocalState(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)
	TestState(t, ls)
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
