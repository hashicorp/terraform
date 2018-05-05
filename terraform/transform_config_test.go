package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestConfigTransformer_nilModule(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	tf := &ConfigTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(g.Vertices()) > 0 {
		t.Fatalf("graph is not empty: %s", g.String())
	}
}

func TestConfigTransformer(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	tf := &ConfigTransformer{Config: testModule(t, "graph-basic")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testConfigTransformerGraphBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_mode(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	tf := &ConfigTransformer{
		Config:     testModule(t, "transform-config-mode-data"),
		ModeFilter: true,
		Mode:       addrs.DataResourceMode,
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
data.aws_ami.foo
`)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_nonUnique(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(NewNodeAbstractResource(
		addrs.RootModuleInstance.Resource(
			addrs.ManagedResourceMode, "aws_instance", "web",
		),
	))
	tf := &ConfigTransformer{Config: testModule(t, "graph-basic")}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.web
aws_instance.web
aws_load_balancer.weblb
aws_security_group.firewall
openstack_floating_ip.random
`)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestConfigTransformer_unique(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(NewNodeAbstractResource(
		addrs.RootModuleInstance.Resource(
			addrs.ManagedResourceMode, "aws_instance", "web",
		),
	))
	tf := &ConfigTransformer{
		Config: testModule(t, "graph-basic"),
		Unique: true,
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.web
aws_load_balancer.weblb
aws_security_group.firewall
openstack_floating_ip.random
`)
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
