package terraform

import (
	"sync"
	"testing"
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
aws_instance.foo[2] - *terraform.NodePlannableResourceInstance
root - terraform.graphNodeRoot
  aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  aws_instance.foo[2] - *terraform.NodePlannableResourceInstance
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
