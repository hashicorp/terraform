package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestNodeDestroyResourceDynamicExpand_deposedCount(t *testing.T) {
	state := mustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
						Provider: "provider.aws",
					},
					"aws_instance.bar.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
						},
						Provider: "provider.aws",
					},
				},
			},
		},
	})

	m := testModule(t, "apply-cbd-count")
	n := &NodeDestroyResourceInstance{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				Addr: addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_instance", "bar",
				),
				Config: m.Module.ManagedResources["aws_instance.bar"],
			},
			InstanceKey:   addrs.IntKey(0),
			ResourceState: state.Modules[""].Resources["aws_instance.bar[0]"],
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   addrs.RootModuleInstance,
		StateState: state.SyncWrapper(),
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(`
aws_instance.bar[0] (deposed 00000001)
`)
	if got != want {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}
