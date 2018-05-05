package terraform

import (
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestNodeDestroyResourceDynamicExpand_deposedCount(t *testing.T) {
	var stateLock sync.RWMutex
	state := &State{
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
					},
					"aws_instance.bar.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
	}

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
			ResourceState: state.Modules[0].Resources["aws_instance.bar.0"],
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   addrs.RootModuleInstance,
		StateState: state,
		StateLock:  &stateLock,
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.bar.0 (deposed #0)
`)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}
