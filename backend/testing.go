package backend

import (
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform/config"
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
func TestBackend(t *testing.T, b Backend) {
	testBackendStates(t, b)
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
	fooState, err := b.State("foo")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := fooState.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := fooState.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	barState, err := b.State("bar")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := barState.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}
	if v := barState.State(); v.HasResources() {
		t.Fatalf("should be empty: %s", v)
	}

	// Verify they are distinct states
	{
		s := barState.State()
		s.Lineage = "bar"
		if err := barState.WriteState(s); err != nil {
			t.Fatalf("bad: %s", err)
		}
		if err := barState.PersistState(); err != nil {
			t.Fatalf("bad: %s", err)
		}

		if err := fooState.RefreshState(); err != nil {
			t.Fatalf("bad: %s", err)
		}
		if v := fooState.State(); v.Lineage == "bar" {
			t.Fatalf("bad: %#v", v)
		}
	}

	// Verify we can now list them
	{
		states, err := b.States()
		if err == ErrNamedStatesNotSupported {
			t.Logf("TestBackend: named states not supported in %T, skipping", b)
			return
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
