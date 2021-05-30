package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

func TestPlanGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(PlanGraphBuilder)
}

func TestPlanGraphBuilder(t *testing.T) {
	awsProvider := &MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Block: simpleTestSchema()},
			ResourceTypes: map[string]providers.Schema{
				"aws_security_group": {Block: simpleTestSchema()},
				"aws_instance":       {Block: simpleTestSchema()},
				"aws_load_balancer":  {Block: simpleTestSchema()},
			},
		},
	}
	openstackProvider := mockProviderWithResourceTypeSchema("openstack_floating_ip", simpleTestSchema())
	components := &basicComponentFactory{
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"):       providers.FactoryFixed(awsProvider),
			addrs.NewDefaultProvider("openstack"): providers.FactoryFixed(openstackProvider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-basic"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[addrs.Provider]*ProviderSchema{
				addrs.NewDefaultProvider("aws"):       awsProvider.ProviderSchema(),
				addrs.NewDefaultProvider("openstack"): openstackProvider.ProviderSchema(),
			},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testPlanGraphBuilderStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPlanGraphBuilder_dynamicBlock(t *testing.T) {
	provider := mockProviderWithResourceTypeSchema("test_thing", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":   {Type: cty.String, Computed: true},
			"list": {Type: cty.List(cty.String), Computed: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	})
	components := &basicComponentFactory{
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): providers.FactoryFixed(provider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-dynblock"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[addrs.Provider]*ProviderSchema{
				addrs.NewDefaultProvider("test"): provider.ProviderSchema(),
			},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	// This test is here to make sure we properly detect references inside
	// the special "dynamic" block construct. The most important thing here
	// is that at the end test_thing.c depends on both test_thing.a and
	// test_thing.b. Other details might shift over time as other logic in
	// the graph builders changes.
	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
meta.count-boundary (EachMode fixup)
  test_thing.c (expand)
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  test_thing.c (expand)
root
  meta.count-boundary (EachMode fixup)
  provider["registry.terraform.io/hashicorp/test"] (close)
test_thing.a (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.b (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.c (expand)
  test_thing.a (expand)
  test_thing.b (expand)
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPlanGraphBuilder_attrAsBlocks(t *testing.T) {
	provider := mockProviderWithResourceTypeSchema("test_thing", &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {Type: cty.String, Computed: true},
			"nested": {
				Type: cty.List(cty.Object(map[string]cty.Type{
					"foo": cty.String,
				})),
				Optional: true,
			},
		},
	})
	components := &basicComponentFactory{
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): providers.FactoryFixed(provider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-attr-as-blocks"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[addrs.Provider]*ProviderSchema{
				addrs.NewDefaultProvider("test"): provider.ProviderSchema(),
			},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	// This test is here to make sure we properly detect references inside
	// the "nested" block that is actually defined in the schema as a
	// list-of-objects attribute. This requires some special effort
	// inside lang.ReferencesInBlock to make sure it searches blocks of
	// type "nested" along with an attribute named "nested".
	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
meta.count-boundary (EachMode fixup)
  test_thing.b (expand)
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  test_thing.b (expand)
root
  meta.count-boundary (EachMode fixup)
  provider["registry.terraform.io/hashicorp/test"] (close)
test_thing.a (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.b (expand)
  test_thing.a (expand)
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPlanGraphBuilder_targetModule(t *testing.T) {
	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-target-module-provider"),
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Logf("Graph: %s", g.String())

	testGraphNotContains(t, g, `module.child1.provider["registry.terraform.io/hashicorp/test"]`)
	testGraphNotContains(t, g, "module.child1.test_object.foo")
}

func TestPlanGraphBuilder_forEach(t *testing.T) {
	awsProvider := mockProviderWithResourceTypeSchema("aws_instance", simpleTestSchema())

	components := &basicComponentFactory{
		providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("aws"): providers.FactoryFixed(awsProvider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "plan-for-each"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[addrs.Provider]*ProviderSchema{
				addrs.NewDefaultProvider("aws"): awsProvider.ProviderSchema(),
			},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	// We're especially looking for the edge here, where aws_instance.bat
	// has a dependency on aws_instance.boo
	expected := strings.TrimSpace(testPlanGraphBuilderForEachStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

const testPlanGraphBuilderStr = `
aws_instance.web (expand)
  aws_security_group.firewall (expand)
  var.foo
aws_load_balancer.weblb (expand)
  aws_instance.web (expand)
aws_security_group.firewall (expand)
  provider["registry.terraform.io/hashicorp/aws"]
local.instance_id (expand)
  aws_instance.web (expand)
meta.count-boundary (EachMode fixup)
  aws_load_balancer.weblb (expand)
  output.instance_id
openstack_floating_ip.random (expand)
  provider["registry.terraform.io/hashicorp/openstack"]
output.instance_id
  local.instance_id (expand)
provider["registry.terraform.io/hashicorp/aws"]
  openstack_floating_ip.random (expand)
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_load_balancer.weblb (expand)
provider["registry.terraform.io/hashicorp/openstack"]
provider["registry.terraform.io/hashicorp/openstack"] (close)
  openstack_floating_ip.random (expand)
root
  meta.count-boundary (EachMode fixup)
  provider["registry.terraform.io/hashicorp/aws"] (close)
  provider["registry.terraform.io/hashicorp/openstack"] (close)
var.foo
`
const testPlanGraphBuilderForEachStr = `
aws_instance.bar (expand)
  provider["registry.terraform.io/hashicorp/aws"]
aws_instance.bar2 (expand)
  provider["registry.terraform.io/hashicorp/aws"]
aws_instance.bat (expand)
  aws_instance.boo (expand)
aws_instance.baz (expand)
  provider["registry.terraform.io/hashicorp/aws"]
aws_instance.boo (expand)
  provider["registry.terraform.io/hashicorp/aws"]
aws_instance.foo (expand)
  provider["registry.terraform.io/hashicorp/aws"]
meta.count-boundary (EachMode fixup)
  aws_instance.bar (expand)
  aws_instance.bar2 (expand)
  aws_instance.bat (expand)
  aws_instance.baz (expand)
  aws_instance.foo (expand)
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_instance.bar (expand)
  aws_instance.bar2 (expand)
  aws_instance.bat (expand)
  aws_instance.baz (expand)
  aws_instance.foo (expand)
root
  meta.count-boundary (EachMode fixup)
  provider["registry.terraform.io/hashicorp/aws"] (close)
`
