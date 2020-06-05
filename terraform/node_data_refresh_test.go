package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/instances"
)

func TestNodeRefreshableDataResourceDynamicExpand_scaleOut(t *testing.T) {
	m := testModule(t, "refresh-data-scale-inout")

	state := MustShimLegacyState(&State{
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
	})

	addr := addrs.RootModule.Resource(addrs.DataResourceMode, "aws_instance", "foo")
	n := &NodeRefreshableDataResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   addr,
			Config: m.Module.DataResources["data.aws_instance.foo"],
		},
		Addr: addr.Absolute(addrs.RootModuleInstance),
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:                 addrs.RootModuleInstance,
		StateState:               state.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(),

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
	m := testModule(t, "refresh-data-scale-inout")

	state := MustShimLegacyState(&State{
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
	})

	addr := addrs.RootModule.Resource(addrs.DataResourceMode, "aws_instance", "foo")
	n := &NodeRefreshableDataResource{
		NodeAbstractResource: &NodeAbstractResource{
			Addr:   addr,
			Config: m.Module.DataResources["data.aws_instance.foo"],
			ResolvedProvider: addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("aws"),
				Module:   addrs.RootModule,
			},
		},
		Addr: addr.Absolute(addrs.RootModuleInstance),
	}

	g, err := n.DynamicExpand(&MockEvalContext{
		PathPath:                 addrs.RootModuleInstance,
		StateState:               state.SyncWrapper(),
		InstanceExpanderExpander: instances.NewExpander(),

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
data.aws_instance.foo[3] - *terraform.NodeDestroyableDataResourceInstance
root - terraform.graphNodeRoot
  data.aws_instance.foo[0] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[1] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[2] - *terraform.NodeRefreshableDataResourceInstance
  data.aws_instance.foo[3] - *terraform.NodeDestroyableDataResourceInstance
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}

	var destroyableDataResource *NodeDestroyableDataResourceInstance
	for _, v := range g.Vertices() {
		if r, ok := v.(*NodeDestroyableDataResourceInstance); ok {
			destroyableDataResource = r
		}
	}

	if destroyableDataResource == nil {
		t.Fatal("failed to find a destroyableDataResource")
	}

	if destroyableDataResource.ResolvedProvider.Provider.Type == "" {
		t.Fatal("NodeDestroyableDataResourceInstance missing provider config")
	}
}
