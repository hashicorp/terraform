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
	deps := []addrs.ConfigResource{
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
	}
	wantDeps := []addrs.ConfigResource{
		addrs.RootModule.Child("child").Resource(addrs.ManagedResourceMode, "test", "flub"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "boop"),
		addrs.RootModule.Resource(addrs.ManagedResourceMode, "test", "honk"),
	}
	rio := &ResourceInstanceObject{
		Value:        value,
		Status:       ObjectPlanned,
		Dependencies: deps,
	}
	rios, err := rio.Encode(value.Type(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(wantDeps, rios.Dependencies); diff != "" {
		t.Errorf("wrong result for deps\n%s", diff)
	}
}
