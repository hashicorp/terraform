package terraform

import (
	"sync"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

func TestNodeRefreshableDataResourceDynamicExpand_scaleOut(t *testing.T) {
	var stateLock sync.RWMutex

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
		NodeAbstractResource: &NodeAbstractResource{
			Addr: addrs.RootModuleInstance.Resource(
				addrs.DataResourceMode,
				"aws_instance",
				"foo",
			),
			Config: m.Module.DataResources["data.aws_instance.foo"],
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   addrs.RootModuleInstance,
		StateState: state,
		StateLock:  &stateLock,

		// DynamicExpand will call EvaluateExpr to evaluate the "count"
		// expression, which is just a literal number 3 in the fixture config
		// and so we'll just hard-code this here too.
		EvaluateExprResult: cty.NumberIntVal(3),
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
		NodeAbstractResource: &NodeAbstractResource{
			Addr: addrs.RootModuleInstance.Resource(
				addrs.DataResourceMode,
				"aws_instance",
				"foo",
			),
			Config: m.Module.DataResources["data.aws_instance.foo"],
		},
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:   addrs.RootModuleInstance,
		StateState: state,
		StateLock:  &stateLock,

		// DynamicExpand will call EvaluateExpr to evaluate the "count"
		// expression, which is just a literal number 3 in the fixture config
		// and so we'll just hard-code this here too.
		EvaluateExprResult: cty.NumberIntVal(3),
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
