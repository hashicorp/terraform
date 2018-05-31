package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
)

func TestPlanGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(PlanGraphBuilder)
}

func TestPlanGraphBuilder(t *testing.T) {
	awsProvider := &MockResourceProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: simpleTestSchema(),
			ResourceTypes: map[string]*configschema.Block{
				"aws_security_group": simpleTestSchema(),
				"aws_instance":       simpleTestSchema(),
				"aws_load_balancer":  simpleTestSchema(),
			},
		},
	}
	openstackProvider := &MockResourceProvider{
		GetSchemaReturn: &ProviderSchema{
			Provider: simpleTestSchema(),
			ResourceTypes: map[string]*configschema.Block{
				"openstack_floating_ip": simpleTestSchema(),
			},
		},
	}
	components := &basicComponentFactory{
		providers: map[string]ResourceProviderFactory{
			"aws":       ResourceProviderFactoryFixed(awsProvider),
			"openstack": ResourceProviderFactoryFixed(openstackProvider),
		},
	}

	b := &PlanGraphBuilder{
		Config:     testModule(t, "graph-builder-plan-basic"),
		Components: components,
		Schemas: &Schemas{
			providers: map[string]*ProviderSchema{
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
  provider.aws
meta.count-boundary (count boundary fixup)
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
  meta.count-boundary (count boundary fixup)
  provider.aws (close)
  provider.openstack (close)
var.foo
`
