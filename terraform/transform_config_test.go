package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config/module"
)

func TestConfigTransformer_nilModule(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(g.Vertices()) > 0 {
		t.Fatalf("graph is not empty: %s", g)
	}
}

func TestConfigTransformer_unloadedModule(t *testing.T) {
	mod, err := module.NewTreeModule(
		"", filepath.Join(fixtureDir, "graph-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformer{Module: mod}
	if err := tf.Transform(&g); err == nil {
		t.Fatal("should error")
	}
}

func TestConfigTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformer{Module: testModule(t, "graph-basic")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testConfigTransformerGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testConfigTransformerGraphBasicStr = `
aws_instance.web
aws_load_balancer.weblb
aws_security_group.firewall
openstack_floating_ip.random
`
