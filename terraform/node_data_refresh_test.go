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
			{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.aws_instance.foo.0": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
								ID: "foo",
							},
						},
					},
					"data.aws_instance.foo.1": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
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
	if err != nil {
		t.Fatalf("error on DynamicExpand: %s", err)
	}

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
			{
				Path: rootModulePath,
				Resources: map[string]*ResourceState{
					"data.aws_instance.foo.0": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
								ID: "foo",
							},
						},
					},
					"data.aws_instance.foo.1": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
								ID: "bar",
							},
						},
					},
					"data.aws_instance.foo.2": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
								ID: "baz",
							},
						},
					},
					"data.aws_instance.foo.3": {
						Type: "aws_instance",
						Deposed: []*InstanceState{
							{
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
	if err != nil {
		t.Fatalf("error on DynamicExpand: %s", err)
	}
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
