// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
)

func TestPlanGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(PlanGraphBuilder)
}

func TestPlanGraphBuilder(t *testing.T) {
	awsProvider := &testing_provider.MockProvider{
		GetProviderSchemaResponse: &providers.GetProviderSchemaResponse{
			Provider: providers.Schema{Body: simpleTestSchema()},
			ResourceTypes: map[string]providers.Schema{
				"aws_security_group": {Body: simpleTestSchema()},
				"aws_instance":       {Body: simpleTestSchema()},
				"aws_load_balancer":  {Body: simpleTestSchema()},
			},
		},
	}
	openstackProvider := mockProviderWithResourceTypeSchema("openstack_floating_ip", simpleTestSchema())
	plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("aws"):       providers.FactoryFixed(awsProvider),
		addrs.NewDefaultProvider("openstack"): providers.FactoryFixed(openstackProvider),
	}, nil, nil)

	b := &PlanGraphBuilder{
		Config:    testModule(t, "graph-builder-plan-basic"),
		Plugins:   plugins,
		Operation: walkPlan,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(testPlanGraphBuilderStr)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
	}

	// We should also be able to derive a graph of the relationships between
	// just the resource addresses, taking into account indirect dependencies
	// through nodes that don't represent resources.
	t.Run("ResourceGraph", func(t *testing.T) {
		resAddrGraph := g.ResourceGraph()
		got := strings.TrimSpace(resAddrGraph.StringForComparison())
		want := strings.TrimSpace(`
aws_instance.web
  aws_security_group.firewall
aws_load_balancer.weblb
  aws_instance.web
aws_security_group.firewall
  openstack_floating_ip.random
openstack_floating_ip.random
`)
		// HINT: aws_security_group.firewall depends on openstack_floating_ip.random
		// because the aws provider configuration refers to it, and all of the
		// aws_-prefixed resource types depend on their provider configuration.
		// We collapse these indirect deps into direct deps as part of lowering
		// into a graph of just resources.
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("wrong result\n%s", diff)
		}

		// Building the resource graph should not have damaged the original graph.
		{
			got := strings.TrimSpace(g.String())
			want := strings.TrimSpace(testPlanGraphBuilderStr)
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("g.ResourceGraph has changed g (should not have modified it)\n%s", diff)
			}
		}
	})
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
	plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): providers.FactoryFixed(provider),
	}, nil, nil)

	b := &PlanGraphBuilder{
		Config:    testModule(t, "graph-builder-plan-dynblock"),
		Plugins:   plugins,
		Operation: walkPlan,
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
	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(`
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  test_thing.c (expand)
root
  provider["registry.terraform.io/hashicorp/test"] (close)
test_thing.a (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.b (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.c (expand)
  test_thing.a (expand)
  test_thing.b (expand)
`)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
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
	plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("test"): providers.FactoryFixed(provider),
	}, nil, nil)

	b := &PlanGraphBuilder{
		Config:    testModule(t, "graph-builder-plan-attr-as-blocks"),
		Plugins:   plugins,
		Operation: walkPlan,
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
	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(`
provider["registry.terraform.io/hashicorp/test"]
provider["registry.terraform.io/hashicorp/test"] (close)
  test_thing.b (expand)
root
  provider["registry.terraform.io/hashicorp/test"] (close)
test_thing.a (expand)
  provider["registry.terraform.io/hashicorp/test"]
test_thing.b (expand)
  test_thing.a (expand)
`)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
	}
}

func TestPlanGraphBuilder_targetModule(t *testing.T) {
	b := &PlanGraphBuilder{
		Config:  testModule(t, "graph-builder-plan-target-module-provider"),
		Plugins: simpleMockPluginLibrary(),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
		Operation: walkPlan,
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

	plugins := newContextPlugins(map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("aws"): providers.FactoryFixed(awsProvider),
	}, nil, nil)

	b := &PlanGraphBuilder{
		Config:    testModule(t, "plan-for-each"),
		Plugins:   plugins,
		Operation: walkPlan,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	got := strings.TrimSpace(g.String())
	// We're especially looking for the edge here, where aws_instance.bat
	// has a dependency on aws_instance.boo
	want := strings.TrimSpace(testPlanGraphBuilderForEachStr)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("wrong result\n%s", diff)
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
openstack_floating_ip.random (expand)
  provider["registry.terraform.io/hashicorp/openstack"]
output.instance_id (expand)
  local.instance_id (expand)
provider["registry.terraform.io/hashicorp/aws"]
  openstack_floating_ip.random (expand)
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_load_balancer.weblb (expand)
provider["registry.terraform.io/hashicorp/openstack"]
provider["registry.terraform.io/hashicorp/openstack"] (close)
  openstack_floating_ip.random (expand)
root
  output.instance_id (expand)
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
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/aws"] (close)
  aws_instance.bar (expand)
  aws_instance.bar2 (expand)
  aws_instance.bat (expand)
  aws_instance.baz (expand)
  aws_instance.foo (expand)
root
  provider["registry.terraform.io/hashicorp/aws"] (close)
`
