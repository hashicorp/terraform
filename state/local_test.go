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

func TestLocalState_conflictUpdate(t *testing.T) {
	ls := testLocalState(t)
	defer os.Remove(ls.Path)

	state := ls.State()
	state.EnsureHasLineage()

	// Writing an unrelated (different lineage) state should work
	// as long as there are no resources in the existing state.
	{
		emptyState := state.DeepCopy()
		emptyState.Modules = []*terraform.ModuleState{
			{
				Path:      []string{"root"},
				Resources: map[string]*terraform.ResourceState{},
			},
		}

		if err := ls.WriteState(emptyState); err != nil {
			t.Fatalf("error %#v while writing empty state; want success", err)
		}

		unrelatedState := emptyState.DeepCopy()
		unrelatedState.Lineage = "--unrelated--"

		if err := ls.WriteState(unrelatedState); err != nil {
			t.Fatalf("error %#v while writing unrelated, empty state; want success", err)
		}
	}

	// On the other hand, writing un unrelated state that *has* resources
	// *should* fail.
	{
		initialState := state.DeepCopy()
		initialState.Modules = []*terraform.ModuleState{
			{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": {},
				},
			},
		}

		if err := ls.WriteState(initialState); err != nil {
			t.Fatalf("error %#v while writing initial state; want success", err)
		}

		unrelatedState := initialState.DeepCopy()
		unrelatedState.Lineage = "--unrelated--"

		if err := ls.WriteState(unrelatedState); err == nil {
			t.Fatalf("success writing unrelated state; want error")
		}
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
