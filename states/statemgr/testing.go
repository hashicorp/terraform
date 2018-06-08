package statemgr

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/terraform/states/statefile"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/states"
)

// TestFull is a helper for testing full state manager implementations. It
// expects that the given implementation is pre-loaded with a snapshot of the
// result from TestFullInitialState.
//
// If the given state manager also implements PersistentMeta, this function
// will test that the snapshot metadata changes as expected between calls
// to the methods of Persistent.
func TestFull(t *testing.T, s Full) {
	t.Helper()

	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Check that the initial state is correct.
	// These do have different Lineages, but we will replace current below.
	initial := TestFullInitialState()
	if state := s.State(); !state.Equal(initial) {
		t.Fatalf("state does not match expected initial state\n\ngot:\n%s\nwant:\n%s", spew.Sdump(state), spew.Sdump(initial))
	}

	var initialMeta SnapshotMeta
	if sm, ok := s.(PersistentMeta); ok {
		initialMeta = sm.StateSnapshotMeta()
	}

	// Now we've proven that the state we're starting with is an initial
	// state, we'll complete our work here with that state, since otherwise
	// further writes would violate the invariant that we only try to write
	// states that share the same lineage as what was initially written.
	current := s.State()

	// Write a new state and verify that we have it
	current.RootModule().SetOutputValue("bar", cty.StringVal("baz"), false)

	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}

	if actual := s.State(); !actual.Equal(current) {
		t.Fatalf("bad:\n%#v\n\n%#v", actual, current)
	}

	// Test persistence
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Refresh if we got it
	if err := s.RefreshState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	var newMeta SnapshotMeta
	if sm, ok := s.(PersistentMeta); ok {
		newMeta = sm.StateSnapshotMeta()
		if got, want := newMeta.Lineage, initialMeta.Lineage; got != want {
			t.Errorf("Lineage changed from %q to %q", want, got)
		}
		if after, before := newMeta.Serial, initialMeta.Serial; after == before {
			t.Errorf("Serial didn't change from %d after new module added", before)
		}
	}

	// Same serial
	serial := newMeta.Serial
	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if sm, ok := s.(PersistentMeta); ok {
		newMeta = sm.StateSnapshotMeta()
		if newMeta.Serial != serial {
			t.Fatalf("serial changed after persisting with no changes: got %d, want %d", newMeta.Serial, serial)
		}
	}

	if sm, ok := s.(PersistentMeta); ok {
		newMeta = sm.StateSnapshotMeta()
	}

	// Change the serial
	current = current.DeepCopy()
	current.EnsureModule(addrs.RootModuleInstance).SetOutputValue(
		"serialCheck", cty.StringVal("true"), false,
	)
	if err := s.WriteState(current); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := s.PersistState(); err != nil {
		t.Fatalf("err: %s", err)
	}

	if sm, ok := s.(PersistentMeta); ok {
		oldMeta := newMeta
		newMeta = sm.StateSnapshotMeta()

		if newMeta.Serial <= serial {
			t.Fatalf("serial incorrect after persisting with changes: got %d, want > %d", newMeta.Serial, serial)
		}

		if newMeta.TerraformVersion != oldMeta.TerraformVersion {
			t.Fatalf("TFVersion changed from %s to %s", oldMeta.TerraformVersion, newMeta.TerraformVersion)
		}

		// verify that Lineage doesn't change along with Serial, or during copying.
		if newMeta.Lineage != oldMeta.Lineage {
			t.Fatalf("Lineage changed from %q to %q", oldMeta.Lineage, newMeta.Lineage)
		}
	}

	// Check that State() returns a copy by modifying the copy and comparing
	// to the current state.
	stateCopy := s.State()
	stateCopy.EnsureModule(addrs.RootModuleInstance.Child("another", addrs.NoKey))
	if reflect.DeepEqual(stateCopy, s.State()) {
		t.Fatal("State() should return a copy")
	}

	// our current expected state should also marshal identically to the persisted state
	if !statefile.StatesMarshalEqual(current, s.State()) {
		t.Fatalf("Persisted state altered unexpectedly.\n\ngot:\n%s\nwant:\n%s", spew.Sdump(s.State()), spew.Sdump(current))
	}
}

// TestFullInitialState is a state that should be snapshotted into a
// full state manager before passing it into TestFull.
func TestFullInitialState() *states.State {
	state := states.NewState()
	childMod := state.EnsureModule(addrs.RootModuleInstance.Child("child", addrs.NoKey))
	rAddr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "foo",
	}
	childMod.SetResourceMeta(rAddr, states.EachList, rAddr.DefaultProviderConfig().Absolute(addrs.RootModuleInstance))
	return state
}
