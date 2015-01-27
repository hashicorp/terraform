package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config/module"
)

func TestConfigTransformer_nilModule(t *testing.T) {
	var g Graph
	tf := &ConfigTransformer{}
	if err := tf.Transform(&g); err == nil {
		t.Fatal("should error")
	}
}

func TestConfigTransformer_unloadedModule(t *testing.T) {
	mod, err := module.NewTreeModule(
		"", filepath.Join(fixtureDir, "graph-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var g Graph
	tf := &ConfigTransformer{Module: mod}
	if err := tf.Transform(&g); err == nil {
		t.Fatal("should error")
	}
}

func TestConfigTransformer(t *testing.T) {
	var g Graph
	tf := &ConfigTransformer{Module: testModule(t, "graph-basic")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_dependsOn(t *testing.T) {
	var g Graph
	tf := &ConfigTransformer{Module: testModule(t, "graph-depends-on")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphDependsOnStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_modules(t *testing.T) {
	var g Graph
	tf := &ConfigTransformer{Module: testModule(t, "graph-modules")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_errMissingDeps(t *testing.T) {
	var g Graph
	tf := &ConfigTransformer{Module: testModule(t, "graph-missing-deps")}
	if err := tf.Transform(&g); err == nil {
		t.Fatalf("err: %s", err)
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
