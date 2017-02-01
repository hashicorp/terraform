package terraform

import (
	"strings"
	"sync"
	"testing"
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

	addr, err := parseResourceAddressInternal("aws_instance.bar.0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	m := testModule(t, "apply-cbd-count")
	n := &NodeDestroyResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:          addr,
			ResourceState: state.Modules[0].Resources["aws_instance.bar.0"],
			Config:        m.Config().Resources[0],
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   []string{"root"},
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
