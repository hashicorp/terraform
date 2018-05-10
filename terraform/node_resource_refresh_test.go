package terraform

import (
	"reflect"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"

	"github.com/davecgh/go-spew/spew"
)

func TestNodeRefreshableManagedResourceDynamicExpand_scaleOut(t *testing.T) {
	var stateLock sync.RWMutex

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
		NodeAbstractResource: &NodeAbstractResource{
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			Config: m.Module.ManagedResources["aws_instance.foo"],
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
		t.Fatalf("error attempting DynamicExpand: %s", err)
	}

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
		NodeAbstractResource: &NodeAbstractResource{
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			Config: m.Module.ManagedResources["aws_instance.foo"],
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
		t.Fatalf("error attempting DynamicExpand: %s", err)
	}
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
	m := testModule(t, "refresh-resource-scale-inout")

	n := &NodeRefreshableManagedResourceInstance{
		NodeAbstractResourceInstance: &NodeAbstractResourceInstance{
			NodeAbstractResource: NodeAbstractResource{
				Addr: addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_instance", "foo",
				),
				Config: m.Module.ManagedResources["aws_instance.foo"],
			},
			InstanceKey: addrs.IntKey(2),
		},
	}

	actual := n.EvalTree()
	expected := n.evalTreeManagedResourceNoState()

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Expected:\n\n%s\nGot:\n\n%s\n", spew.Sdump(expected), spew.Sdump(actual))
	}
}
