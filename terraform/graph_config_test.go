package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config/module"
)

func TestGraph_nilModule(t *testing.T) {
	_, err := Graph2(nil)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestGraph_unloadedModule(t *testing.T) {
	mod, err := module.NewTreeModule(
		"", filepath.Join(fixtureDir, "graph-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := Graph2(mod); err == nil {
		t.Fatal("should error")
	}
}

func TestGraph(t *testing.T) {
	g, err := Graph2(testModule(t, "graph-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraph2_dependsOn(t *testing.T) {
	g, err := Graph2(testModule(t, "graph-depends-on"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphDependsOnStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraph2_modules(t *testing.T) {
	g, err := Graph2(testModule(t, "graph-modules"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraph2_errMissingDeps(t *testing.T) {
	_, err := Graph2(testModule(t, "graph-missing-deps"))
	if err == nil {
		t.Fatal("should error")
	}
}

const testGraphBasicStr = `
aws_instance.web
  aws_security_group.firewall
aws_load_balancer.weblb
  aws_instance.web
aws_security_group.firewall
openstack_floating_ip.random
provider.aws
  openstack_floating_ip.random
`

const testGraphDependsOnStr = `
aws_instance.db
  aws_instance.web
aws_instance.web
`

const testGraphModulesStr = `
aws_instance.web
  aws_security_group.firewall
  module.consul
aws_security_group.firewall
module.consul
  aws_security_group.firewall
provider.aws
`
