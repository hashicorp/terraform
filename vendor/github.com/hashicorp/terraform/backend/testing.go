package backend

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// TestBackendConfig validates and configures the backend with the
// given configuration.
func TestBackendConfig(t *testing.T, b Backend, c map[string]interface{}) Backend {
	// Get the proper config structure
	rc, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}
	conf := terraform.NewResourceConfig(rc)

	// Validate
	warns, errs := b.Validate(conf)
	if len(warns) > 0 {
		t.Fatalf("warnings: %s", warns)
	}
	if len(errs) > 0 {
		t.Fatalf("errors: %s", errs)
	}

	// Configure
	if err := b.Configure(conf); err != nil {
		t.Fatalf("err: %s", err)
	}

	return b
}

// TestBackend will test the functionality of a Backend. The backend is
// assumed to already be configured. This will test state functionality.
// If the backend reports it doesn't support multi-state by returning the
// error ErrNamedStatesNotSupported, then it will not test that.
//
// If you want to test locking, two backends must be given. If b2 is nil,
// then state lockign won't be tested.
func TestBackend(t *testing.T, b1, b2 Backend) {
	testBackendStates(t, b1)

	if b2 != nil {
		testBackendStateLock(t, b1, b2)
	}
}

func testBackendStates(t *testing.T, b Backend) {
	states, err := b.States()
	if err == ErrNamedStatesNotSupported {
		t.Logf("TestBackend: named states not supported in %T, skipping", b)
		return
	}

	// Test it starts with only the default
	if len(states) != 1 || states[0] != DefaultStateName {
		t.Fatalf("should only have default to start: %#v", states)
	}

	// Create a couple states
	foo, err := b.State("foo")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := foo.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := foo.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	bar, err := b.State("bar")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := bar.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := bar.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	// Verify they are distinct states that can be read back from storage
	{
		// start with a fresh state, and record the lineage being
		// written to "bar"
		barState := terraform.NewState()

		// creating the named state may have created a lineage, so use that if it exists.
		if s := bar.State(); s != nil && s.Lineage != "" {
			barState.Lineage = s.Lineage
		}
		barLineage := barState.Lineage

		// the foo lineage should be distinct from bar, and unchanged after
		// modifying bar
		fooState := terraform.NewState()
		// creating the named state may have created a lineage, so use that if it exists.
		if s := foo.State(); s != nil && s.Lineage != "" {
			fooState.Lineage = s.Lineage
		}
		fooLineage := fooState.Lineage

		// write a known state to foo
		if err := foo.WriteState(fooState); err != nil {
			t.Fatal("error writing foo state:", err)
		}
		if err := foo.PersistState(); err != nil {
			t.Fatal("error persisting foo state:", err)
		}

		// write a distinct known state to bar
		if err := bar.WriteState(barState); err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := bar.PersistState(); err != nil {
			t.Fatalf("bad: %s", err)
		}

		// verify that foo is unchanged with the existing state manager
		if err := foo.RefreshState(); err != nil {
			t.Fatal("error refreshing foo:", err)
		}
		fooState = foo.State()
		switch {
		case fooState == nil:
			t.Fatal("nil state read from foo")
		case fooState.Lineage == barLineage:
			t.Fatalf("bar lineage read from foo: %#v", fooState)
		case fooState.Lineage != fooLineage:
			t.Fatal("foo lineage alterred")
		}

		// fetch foo again from the backend
		foo, err = b.State("foo")
		if err != nil {
			t.Fatal("error re-fetching state:", err)
		}
		if err := foo.RefreshState(); err != nil {
			t.Fatal("error refreshing foo:", err)
		}
		fooState = foo.State()
		switch {
		case fooState == nil:
			t.Fatal("nil state read from foo")
		case fooState.Lineage != fooLineage:
			t.Fatal("incorrect state returned from backend")
		}

		// fetch the bar  again from the backend
		bar, err = b.State("bar")
		if err != nil {
			t.Fatal("error re-fetching state:", err)
		}
		if err := bar.RefreshState(); err != nil {
			t.Fatal("error refreshing bar:", err)
		}
		barState = bar.State()
		switch {
		case barState == nil:
			t.Fatal("nil state read from bar")
		case barState.Lineage != barLineage:
			t.Fatal("incorrect state returned from backend")
		}
	}

	// Verify we can now list them
	{
		// we determined that named stated are supported earlier
		states, err := b.States()
		if err != nil {
			t.Fatal(err)
		}

		sort.Strings(states)
		expected := []string{"bar", "default", "foo"}
		if !reflect.DeepEqual(states, expected) {
			t.Fatalf("bad: %#v", states)
		}
	}

	// Delete some states
	if err := b.DeleteState("foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the default state can't be deleted
	if err := b.DeleteState(DefaultStateName); err == nil {
		t.Fatal("expected error")
	}

	// Create and delete the foo state again.
	// Make sure that there are no leftover artifacts from a deleted state
	// preventing re-creation.
	foo, err = b.State("foo")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := foo.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := foo.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}
	// and delete it again
	if err := b.DeleteState("foo"); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify deletion
	{
		states, err := b.States()
		if err == ErrNamedStatesNotSupported {
			t.Logf("TestBackend: named states not supported in %T, skipping", b)
			return
		}

		sort.Strings(states)
		expected := []string{"bar", "default"}
		if !reflect.DeepEqual(states, expected) {
			t.Fatalf("bad: %#v", states)
		}
	}
}

func testBackendStateLock(t *testing.T, b1, b2 Backend) {
	// Get the default state for each
	b1StateMgr, err := b1.State(DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b1StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Fast exit if this doesn't support locking at all
	if _, ok := b1StateMgr.(state.Locker); !ok {
		t.Logf("TestBackend: backend %T doesn't support state locking, not testing", b1)
		return
	}

	t.Logf("TestBackend: testing state locking for %T", b1)

	b2StateMgr, err := b2.State(DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b2StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Reassign so its obvious whats happening
	lockerA := b1StateMgr.(state.Locker)
	lockerB := b2StateMgr.(state.Locker)

	infoA := state.NewLockInfo()
	infoA.Operation = "test"
	infoA.Who = "clientA"

	infoB := state.NewLockInfo()
	infoB.Operation = "test"
	infoB.Who = "clientB"

	lockIDA, err := lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// If the lock ID is blank, assume locking is disabled
	if lockIDA == "" {
		t.Logf("TestBackend: %T: empty string returned for lock, assuming disabled", b1)
		return
	}

	_, err = lockerB.Lock(infoB)
	if err == nil {
		lockerA.Unlock(lockIDA)
		t.Fatal("client B obtained lock while held by client A")
	}

	if err := lockerA.Unlock(lockIDA); err != nil {
		t.Fatal("error unlocking client A", err)
	}

	lockIDB, err := lockerB.Lock(infoB)
	if err != nil {
		t.Fatal("unable to obtain lock from client B")
	}

	if lockIDB == lockIDA {
		t.Fatalf("duplicate lock IDs: %q", lockIDB)
	}

	if err = lockerB.Unlock(lockIDB); err != nil {
		t.Fatal("error unlocking client B:", err)
	}

}
