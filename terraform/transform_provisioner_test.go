package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestMissingProvisionerTransformer(t *testing.T) {
	mod := testModule(t, "transform-provisioner-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProvisionerTransformer{Provisioners: []string{"foo"}}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProvisionerTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProvisionerBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneProvisionerTransformer(t *testing.T) {
	mod := testModule(t, "transform-provisioner-prune")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProvisionerTransformer{
			Provisioners: []string{"foo", "bar"}}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ProvisionerTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProvisionerTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &PruneProvisionerTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneProvisionerBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeMissingProvisioner_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeMissingProvisioner)
	var _ dag.NamedVertex = new(graphNodeMissingProvisioner)
	var _ GraphNodeProvisioner = new(graphNodeMissingProvisioner)
}

func TestGraphNodeMissingProvisioner_ProvisionerName(t *testing.T) {
	n := &graphNodeMissingProvisioner{ProvisionerNameValue: "foo"}
	if v := n.ProvisionerName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformMissingProvisionerBasicStr = `
aws_instance.web
provisioner.foo
provisioner.shell (close)
  aws_instance.web
`

const testTransformPruneProvisionerBasicStr = `
aws_instance.web
  provisioner.foo
provisioner.foo
provisioner.foo (close)
  aws_instance.web
`
