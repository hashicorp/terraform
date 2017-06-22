package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNodeRefreshableManagedResourceDynamicExpand_scaleOut(t *testing.T) {
	var stateLock sync.RWMutex

	addr, err := ParseResourceAddress("aws_instance.foo")
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	m := testModule(t, "refresh-resource-scale-inout")

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
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

	n := &NodeRefreshableManagedResource{
		NodeAbstractCountResource: &NodeAbstractCountResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr:   addr,
				Config: m.Config().Resources[0],
			},
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   []string{"root"},
		StateState: state,
		StateLock:  &stateLock,
	})

	actual := g.StringWithNodeTypes()
	expected := `aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
root - terraform.graphNodeRoot
  aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}

func TestNodeRefreshableManagedResourceDynamicExpand_scaleIn(t *testing.T) {
	var stateLock sync.RWMutex

	addr, err := ParseResourceAddress("aws_instance.foo")
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	m := testModule(t, "refresh-resource-scale-inout")

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
						},
					},
					"aws_instance.foo.2": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "baz",
							},
						},
					},
					"aws_instance.foo.3": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "qux",
							},
						},
					},
				},
			},
		},
	}

	n := &NodeRefreshableManagedResource{
		NodeAbstractCountResource: &NodeAbstractCountResource{
			NodeAbstractResource: &NodeAbstractResource{
				Addr:   addr,
				Config: m.Config().Resources[0],
			},
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   []string{"root"},
		StateState: state,
		StateLock:  &stateLock,
	})

	actual := g.StringWithNodeTypes()
	expected := `aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
aws_instance.foo[3] - *terraform.NodeRefreshableManagedResourceInstance
root - terraform.graphNodeRoot
  aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[3] - *terraform.NodeRefreshableManagedResourceInstance
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}

func TestNodeRefreshableManagedResourceEvalTree_scaleOut(t *testing.T) {
	var provider ResourceProvider
	var state *InstanceState
	var resourceConfig *ResourceConfig

	addr, err := ParseResourceAddress("aws_instance.foo[2]")
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	m := testModule(t, "refresh-resource-scale-inout")

	n := &NodeRefreshableManagedResourceInstance{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   addr,
			Config: m.Config().Resources[0],
		},
	}

	actual := n.EvalTree()

	expected := &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.Config.RawConfig.Copy(),
				Resource: &Resource{
					Name:       addr.Name,
					Type:       addr.Type,
					CountIndex: addr.Index,
				},
				Output: &resourceConfig,
			},
			&EvalGetProvider{
				Name:   n.ProvidedBy()[0],
				Output: &provider,
			},
			&EvalValidateResource{
				Provider:       &provider,
				Config:         &resourceConfig,
				ResourceName:   n.Config.Name,
				ResourceType:   n.Config.Type,
				ResourceMode:   n.Config.Mode,
				IgnoreWarnings: true,
			},
			&EvalReadState{
				Name:   addr.stateId(),
				Output: &state,
			},
			&EvalDiff{
				Name: addr.stateId(),
				Info: &InstanceInfo{
					Id:         addr.stateId(),
					Type:       addr.Type,
					ModulePath: normalizeModulePath(addr.Path),
				},
				Config:      &resourceConfig,
				Resource:    n.Config,
				Provider:    &provider,
				State:       &state,
				OutputState: &state,
				Stub:        true,
			},
			&EvalWriteState{
				Name:         addr.stateId(),
				ResourceType: n.Config.Type,
				Provider:     n.Config.Provider,
				Dependencies: n.StateReferences(),
				State:        &state,
			},
		},
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected:\n\n%s\nGot:\n\n%s\n", spew.Sdump(expected), spew.Sdump(actual))
	}
}
