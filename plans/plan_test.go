package plans

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestProviderAddrs(t *testing.T) {

	plan := &Plan{
		VariableValues: map[string]DynamicValue{},
		Changes: &Changes{
			Resources: []*ResourceInstanceChangeSrc{
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "woot",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					DeposedKey: "foodface",
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule,
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
				{
					Addr: addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "what",
					}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
					ProviderAddr: addrs.AbsProviderConfig{
						Module:   addrs.RootModule.Child("foo"),
						Provider: addrs.NewDefaultProvider("test"),
					},
				},
			},
		},
	}

	got := plan.ProviderAddrs()
	want := []addrs.AbsProviderConfig{
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule.Child("foo"),
			Provider: addrs.NewDefaultProvider("test"),
		},
		addrs.AbsProviderConfig{
			Module:   addrs.RootModule,
			Provider: addrs.NewDefaultProvider("test"),
		},
	}

	for _, problem := range deep.Equal(got, want) {
		t.Error(problem)
	}
}

// Module outputs should not effect the result of Empty
func TestModuleOutputChangesEmpty(t *testing.T) {
	changes := &Changes{
		Outputs: []*OutputChangeSrc{
			{
				Addr: addrs.AbsOutputValue{
					Module: addrs.RootModuleInstance.Child("child", addrs.NoKey),
					OutputValue: addrs.OutputValue{
						Name: "output",
					},
				},
				ChangeSrc: ChangeSrc{
					Action: Update,
					Before: []byte("a"),
					After:  []byte("b"),
				},
			},
		},
	}

	if !changes.Empty() {
		t.Fatal("plan has no visible changes")
	}
}
