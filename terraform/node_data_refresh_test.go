package terraform

import (
	"sync"
	"testing"
)

func TestNodeRefreshableDataResourceDynamicExpand_scaleOut(t *testing.T) {
	var stateLock sync.RWMutex

	addr, err := ParseResourceAddress("data.aws_instance.foo")
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	m := testModule(t, "refresh-data-scale-inout")

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"data.aws_instance.foo.1": &ResourceState{
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

	n := &NodeRefreshableDataResource{
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
	expected := `data.aws_instance.foo[0] - *terraform.NodeRefreshableDataResourceInstance
data.aws_instance.foo[1] - *terraform.NodeRefreshableDataResourceInstance
data.aws_instance.foo[2] - *terraform.NodeRefreshableDataResourceInstance
root - terraform.graphNodeRoot
  data.aws_instance.foo[0] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[1] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[2] - *terraform.NodeRefreshableDataResourceInstance
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}

func TestNodeRefreshableDataResourceDynamicExpand_scaleIn(t *testing.T) {
	var stateLock sync.RWMutex

	addr, err := ParseResourceAddress("data.aws_instance.foo")
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	m := testModule(t, "refresh-data-scale-inout")

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "foo",
							},
						},
					},
					"data.aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "bar",
							},
						},
					},
					"data.aws_instance.foo.2": &ResourceState{
						Type: "aws_instance",
						Deposed: []*InstanceState{
							&InstanceState{
								ID: "baz",
							},
						},
					},
					"data.aws_instance.foo.3": &ResourceState{
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

	n := &NodeRefreshableDataResource{
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
	expected := `data.aws_instance.foo[0] - *terraform.NodeRefreshableDataResourceInstance
data.aws_instance.foo[1] - *terraform.NodeRefreshableDataResourceInstance
data.aws_instance.foo[2] - *terraform.NodeRefreshableDataResourceInstance
data.aws_instance.foo[3] - *terraform.NodeDestroyableDataResource
root - terraform.graphNodeRoot
  data.aws_instance.foo[0] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[1] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[2] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[3] - *terraform.NodeDestroyableDataResource
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}
