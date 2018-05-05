package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestRefreshGraphBuilder_configOrphans(t *testing.T) {

	m := testModule(t, "refresh-config-orphan")

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
				},
			},
		},
	}

	b := &RefreshGraphBuilder{
		Config: m,
		State:  state,
		Components: &basicComponentFactory{
			providers: map[string]ResourceProviderFactory{
				"aws": func() (ResourceProvider, error) {
					return nil, fmt.Errorf("not implemented")
				},
			},
		},
	}
	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("Error building graph: %s", err)
	}

	actual := g.StringWithNodeTypes()
	expected := `aws_instance.foo - *terraform.NodeRefreshableManagedResource
  provider.aws - *terraform.NodeApplyableProvider
data.aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  provider.aws - *terraform.NodeApplyableProvider
data.aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  provider.aws - *terraform.NodeApplyableProvider
data.aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
  provider.aws - *terraform.NodeApplyableProvider
provider.aws - *terraform.NodeApplyableProvider
provider.aws (close) - *terraform.graphNodeCloseProvider
  aws_instance.foo - *terraform.NodeRefreshableManagedResource
  data.aws_instance.foo[0] - *terraform.NodeRefreshableManagedResourceInstance
  data.aws_instance.foo[1] - *terraform.NodeRefreshableManagedResourceInstance
  data.aws_instance.foo[2] - *terraform.NodeRefreshableManagedResourceInstance
`
	if expected != actual {
		t.Fatalf("Expected:\n%s\nGot:\n%s", expected, actual)
	}
}
