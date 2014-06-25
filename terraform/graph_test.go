package terraform

import (
	"strings"
	"testing"
)

func TestGraph(t *testing.T) {
	config := testConfig(t, "graph-basic")

	g := Graph(config, nil)
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTerraformGraphStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraph_cycle(t *testing.T) {
	config := testConfig(t, "graph-cycle")

	g := Graph(config, nil)
	if err := g.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestGraph_state(t *testing.T) {
	config := testConfig(t, "graph-basic")
	state := &State{
		Resources: map[string]*ResourceState{
			"aws_instance.old": &ResourceState{
				ID:   "foo",
				Type: "aws_instance",
			},
		},
	}

	g := Graph(config, state)
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTerraformGraphStateStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphFull(t *testing.T) {
	rpAws := new(MockResourceProvider)
	rpOS := new(MockResourceProvider)

	rpAws.ResourcesReturn = []ResourceType{
		ResourceType{Name: "aws_instance"},
		ResourceType{Name: "aws_load_balancer"},
		ResourceType{Name: "aws_security_group"},
	}
	rpOS.ResourcesReturn = []ResourceType{
		ResourceType{Name: "openstack_floating_ip"},
	}

	ps := map[string]ResourceProviderFactory{
		"aws":  testProviderFuncFixed(rpAws),
		"open": testProviderFuncFixed(rpOS),
	}

	c := testConfig(t, "graph-basic")
	g := Graph(c, nil)
	if err := GraphFull(g, ps); err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	// A helper to help get us the provider for a resource.
	graphProvider := func(n string) ResourceProvider {
		return g.Noun(n).Meta.(*GraphNodeResource).Resource.Provider
	}

	// Test a couple
	if graphProvider("aws_instance.web") != rpAws {
		t.Fatalf("bad: %#v", graphProvider("aws_instance.web"))
	}
	if graphProvider("openstack_floating_ip.random") != rpOS {
		t.Fatalf("bad: %#v", graphProvider("openstack_floating_ip.random"))
	}

	// Test that all providers have been set
	for _, n := range g.Nouns {
		switch m := n.Meta.(type) {
		case *GraphNodeResource:
			if m.Resource.Provider == nil {
				t.Fatalf("bad: %#v", m)
			}
		case *GraphNodeResourceProvider:
			if len(m.Providers) == 0 {
				t.Fatalf("bad: %#v", m)
			}
		default:
			continue
		}
	}
}

const testTerraformGraphStr = `
root: root
aws_instance.web
  aws_instance.web -> aws_security_group.firewall
  aws_instance.web -> provider.aws
aws_load_balancer.weblb
  aws_load_balancer.weblb -> aws_instance.web
  aws_load_balancer.weblb -> provider.aws
aws_security_group.firewall
  aws_security_group.firewall -> provider.aws
openstack_floating_ip.random
provider.aws
  provider.aws -> openstack_floating_ip.random
root
  root -> aws_instance.web
  root -> aws_load_balancer.weblb
  root -> aws_security_group.firewall
  root -> openstack_floating_ip.random
  root -> provider.aws
`

const testTerraformGraphStateStr = `
root: root
aws_instance.old
  aws_instance.old -> provider.aws
aws_instance.web
  aws_instance.web -> aws_security_group.firewall
  aws_instance.web -> provider.aws
aws_load_balancer.weblb
  aws_load_balancer.weblb -> aws_instance.web
  aws_load_balancer.weblb -> provider.aws
aws_security_group.firewall
  aws_security_group.firewall -> provider.aws
openstack_floating_ip.random
provider.aws
  provider.aws -> openstack_floating_ip.random
root
  root -> aws_instance.old
  root -> aws_instance.web
  root -> aws_load_balancer.weblb
  root -> aws_security_group.firewall
  root -> openstack_floating_ip.random
  root -> provider.aws
`
