package plans

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/addrs"
)

func TestProviderAddrs(t *testing.T) {

	plan := &Plan{
		VariableValues: map[string]DynamicValue{},
		Changes: &Changes{
			RootOutputs: map[string]*OutputChange{},
			Resources: []*ResourceInstanceChange{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance),
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					DeposedKey: "foodface",
					ProviderAddr: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance),
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "what",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.ProviderConfig{
						Type: "test",
					}.Absolute(addrs.RootModuleInstance.Child("foo", addrs.NoKey)),
				},
			},
		},
	}

	got := plan.ProviderAddrs()
	want := []addrs.AbsProviderConfig{
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance.Child("foo", addrs.NoKey)),
		addrs.ProviderConfig{
			Type: "test",
		}.Absolute(addrs.RootModuleInstance),
	}

	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}
