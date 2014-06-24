package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestConfigGraph(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "resource_graph.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	graph := c.Graph()
	if err := graph.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(graph.String())
	expected := resourceGraphValue

	if actual != strings.TrimSpace(expected) {
		t.Fatalf("bad:\n%s", actual)
	}
}

func TestConfigGraph_cycle(t *testing.T) {
	c, err := Load(filepath.Join(fixtureDir, "resource_graph_cycle.tf"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	graph := c.Graph()
	if err := graph.Validate(); err == nil {
		t.Fatal("graph should be invalid")
	}
}

func TestNewResourceVariable(t *testing.T) {
	v, err := NewResourceVariable("foo.bar.baz")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Type != "foo" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v)
	}
	if v.Field != "baz" {
		t.Fatalf("bad: %#v", v)
	}

	if v.FullKey() != "foo.bar.baz" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestNewUserVariable(t *testing.T) {
	v, err := NewUserVariable("var.bar")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if v.Name != "bar" {
		t.Fatalf("bad: %#v", v.Name)
	}
	if v.FullKey() != "var.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestProviderConfigName(t *testing.T) {
	pcs := map[string]*ProviderConfig{
		"aw":   new(ProviderConfig),
		"aws":  new(ProviderConfig),
		"a":    new(ProviderConfig),
		"gce_": new(ProviderConfig),
	}

	n := ProviderConfigName("aws_instance", pcs)
	if n != "aws" {
		t.Fatalf("bad: %s", n)
	}
}

const resourceGraphValue = `
root: root
openstack_floating_ip.random
aws_security_group.firewall
  aws_security_group.firewall -> provider.aws
aws_instance.web
  aws_instance.web -> aws_security_group.firewall
  aws_instance.web -> provider.aws
aws_load_balancer.weblb
  aws_load_balancer.weblb -> aws_instance.web
  aws_load_balancer.weblb -> provider.aws
provider.aws
  provider.aws -> openstack_floating_ip.random
root
  root -> openstack_floating_ip.random
  root -> aws_security_group.firewall
  root -> aws_instance.web
  root -> aws_load_balancer.weblb
  root -> provider.aws
`
