package states

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestResourceInstanceObject_encode(t *testing.T) {
	value := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.True,
	})
	// The in-memory order of resource dependencies is random, since they're an
	// unordered set.
	depsOne := []addrs.ConfigResource{
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
	}
	depsTwo := []addrs.ConfigResource{
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
	}
	rioOne := &ResourceInstanceObject{
		Value:        value,
		Status:       ObjectPlanned,
		Dependencies: depsOne,
	}
	rioTwo := &ResourceInstanceObject{
		Value:        value,
		Status:       ObjectPlanned,
		Dependencies: depsTwo,
	}
	riosOne, err := rioOne.Encode(value.Type(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	riosTwo, err := rioTwo.Encode(value.Type(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// However, identical sets of dependencies should always be written to state
	// in an identical order, so we don't do meaningless state updates on refresh.
	if diff := cmp.Diff(riosOne.Dependencies, riosTwo.Dependencies); diff != "" {
		t.Errorf("identical dependencies got encoded in different orders:\n%s", diff)
	}
}
