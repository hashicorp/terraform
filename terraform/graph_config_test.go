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
