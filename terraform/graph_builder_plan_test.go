package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/zclconf/go-cty/cty"
)

func TestPlanGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(PlanGraphBuilder)
}

func TestPlanGraphBuilder(t *testing.T) {
	awsProvider := &MockProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: simpleTestSchema(),
			ResourceTypes: map[string]*configschema.Block{
				"aws_security_group": simpleTestSchema(),
				"aws_instance":       simpleTestSchema(),
				"aws_load_balancer":  simpleTestSchema(),
			},
		},
	}
	openstackProvider := &MockProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: simpleTestSchema(),
			ResourceTypes: map[string]*configschema.Block{
				"openstack_floating_ip": simpleTestSchema(),
			},
		},
	}
	components := &basicComponentFactory{
		providers: map[string]providers.Factory{
			"aws":       providers.FactoryFixed(awsProvider),
			"openstack": providers.FactoryFixed(openstackProvider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-basic"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[string]*ProviderSchema{
				"aws":       awsProvider.GetSchemaReturn,
				"openstack": openstackProvider.GetSchemaReturn,
			},
		},
		DisableReduce: true,
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
	provider := &MockProvider{
		GetSchemaReturn: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"test_thing": {
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
				},
			},
		},
	}
	components := &basicComponentFactory{
		providers: map[string]providers.Factory{
			"test": providers.FactoryFixed(provider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-dynblock"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[string]*ProviderSchema{
				"test": provider.GetSchemaReturn,
			},
		},
		DisableReduce: true,
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
  provider.test
  test_thing.a
  test_thing.b
  test_thing.c
provider.test
provider.test (close)
  provider.test
  test_thing.a
  test_thing.b
  test_thing.c
root
  meta.count-boundary (EachMode fixup)
  provider.test (close)
test_thing.a
  provider.test
test_thing.b
  provider.test
test_thing.c
  provider.test
  test_thing.a
  test_thing.b
`)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPlanGraphBuilder_attrAsBlocks(t *testing.T) {
	provider := &MockProvider{
		GetSchemaReturn: &ProviderSchema{
			ResourceTypes: map[string]*configschema.Block{
				"test_thing": {
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Computed: true},
						"nested": {
							Type: cty.List(cty.Object(map[string]cty.Type{
								"foo": cty.String,
							})),
							Optional: true,
						},
					},
				},
			},
		},
	}
	components := &basicComponentFactory{
		providers: map[string]providers.Factory{
			"test": providers.FactoryFixed(provider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-attr-as-blocks"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[string]*ProviderSchema{
				"test": provider.GetSchemaReturn,
			},
		},
		DisableReduce: true,
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
  provider.test
  test_thing.a
  test_thing.b
provider.test
provider.test (close)
  provider.test
  test_thing.a
  test_thing.b
root
  meta.count-boundary (EachMode fixup)
  provider.test (close)
test_thing.a
  provider.test
test_thing.b
  provider.test
  test_thing.a
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

	testGraphNotContains(t, g, "module.child1.provider.test")
	testGraphNotContains(t, g, "module.child1.test_object.foo")
}

func TestPlanGraphBuilder_forEach(t *testing.T) {
	awsProvider := &MockProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: simpleTestSchema(),
			ResourceTypes: map[string]*configschema.Block{
				"aws_instance": simpleTestSchema(),
			},
		},
	}

	components := &basicComponentFactory{
		providers: map[string]providers.Factory{
			"aws": providers.FactoryFixed(awsProvider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "plan-for-each"),
		Components: components,
		Schemas: &Schemas{
			Providers: map[string]*ProviderSchema{
				"aws": awsProvider.GetSchemaReturn,
			},
		},
		DisableReduce: true,
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
aws_instance.web
  aws_security_group.firewall
  provider.aws
  var.foo
aws_load_balancer.weblb
  aws_instance.web
  provider.aws
aws_security_group.firewall
  provider.aws
local.instance_id
  aws_instance.web
meta.count-boundary (EachMode fixup)
  aws_instance.web
  aws_load_balancer.weblb
  aws_security_group.firewall
  local.instance_id
  openstack_floating_ip.random
  output.instance_id
  provider.aws
  provider.openstack
  var.foo
openstack_floating_ip.random
  provider.openstack
output.instance_id
  local.instance_id
provider.aws
  openstack_floating_ip.random
provider.aws (close)
  aws_instance.web
  aws_load_balancer.weblb
  aws_security_group.firewall
  provider.aws
provider.openstack
provider.openstack (close)
  openstack_floating_ip.random
  provider.openstack
root
  meta.count-boundary (EachMode fixup)
  provider.aws (close)
  provider.openstack (close)
var.foo
`
const testPlanGraphBuilderForEachStr = `
aws_instance.bar
  provider.aws
aws_instance.bar2
  provider.aws
aws_instance.bat
  aws_instance.boo
  provider.aws
aws_instance.baz
  provider.aws
aws_instance.boo
  provider.aws
aws_instance.foo
  provider.aws
meta.count-boundary (EachMode fixup)
  aws_instance.bar
  aws_instance.bar2
  aws_instance.bat
  aws_instance.baz
  aws_instance.boo
  aws_instance.foo
  provider.aws
provider.aws
provider.aws (close)
  aws_instance.bar
  aws_instance.bar2
  aws_instance.bat
  aws_instance.baz
  aws_instance.boo
  aws_instance.foo
  provider.aws
root
  meta.count-boundary (EachMode fixup)
  provider.aws (close)
`
