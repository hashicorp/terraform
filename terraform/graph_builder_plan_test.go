package terraform

import (
	"reflect"
	"strings"
	"testing"
)

func TestPlanGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(PlanGraphBuilder)
}

func TestPlanGraphBuilder(t *testing.T) {
	b := &PlanGraphBuilder{
		Module:        testModule(t, "graph-builder-plan-basic"),
		Providers:     []string{"aws", "openstack"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testPlanGraphBuilderStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

func TestPlanGraphBuilder_targetModule(t *testing.T) {
	b := &PlanGraphBuilder{
		Module:    testModule(t, "graph-builder-plan-target-module-provider"),
		Providers: []string{"null"},
		Targets:   []string{"module.child2"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Logf("Graph: %s", g.String())

	testGraphNotContains(t, g, "module.child1.provider.null")
	testGraphNotContains(t, g, "module.child1.null_resource.foo")
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
meta.count-boundary (count boundary fixup)
  aws_instance.web
  aws_load_balancer.weblb
  aws_security_group.firewall
  openstack_floating_ip.random
  provider.aws
  provider.openstack
  var.foo
openstack_floating_ip.random
  provider.openstack
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
