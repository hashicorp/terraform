package terraform

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/config/module"
)

func TestConfigTransformerOld_nilModule(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{}
	if err := tf.Transform(&g); err == nil {
		t.Fatal("should error")
	}
}

func TestConfigTransformerOld_unloadedModule(t *testing.T) {
	mod, err := module.NewTreeModule(
		"", filepath.Join(fixtureDir, "graph-basic"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: mod}
	if err := tf.Transform(&g); err == nil {
		t.Fatal("should error")
	}
}

func TestConfigTransformerOld(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-basic")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformerOld_dependsOn(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-depends-on")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphDependsOnStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformerOld_modules(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-modules")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformerOld_outputs(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-outputs")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphOutputsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformerOld_providerAlias(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-provider-alias")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphProviderAliasStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformerOld_errMissingDeps(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &ConfigTransformerOld{Module: testModule(t, "graph-missing-deps")}
	if err := tf.Transform(&g); err == nil {
		t.Fatalf("err: %s", err)
	}
}

const testGraphBasicStr = `
aws_instance.web
  aws_security_group.firewall
  var.foo
aws_load_balancer.weblb
  aws_instance.web
aws_security_group.firewall
openstack_floating_ip.random
provider.aws
  openstack_floating_ip.random
var.foo
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

const testGraphOutputsStr = `
aws_instance.foo
output.foo
  aws_instance.foo
`

const testGraphProviderAliasStr = `
provider.aws
provider.aws.bar
provider.aws.foo
`
