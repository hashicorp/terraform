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
		transform := &MissingProvisionerTransformer{Provisioners: []string{"shell"}}
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

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProvisionerBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCloseProvisionerTransformer(t *testing.T) {
	mod := testModule(t, "transform-provisioner-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProvisionerTransformer{Provisioners: []string{"shell"}}
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

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCloseProvisionerBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}
func TestGraphNodeProvisioner_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeProvisioner)
	var _ dag.NamedVertex = new(graphNodeProvisioner)
	var _ GraphNodeProvisioner = new(graphNodeProvisioner)
}

func TestGraphNodeProvisioner_ProvisionerName(t *testing.T) {
	n := &graphNodeProvisioner{ProvisionerNameValue: "foo"}
	if v := n.ProvisionerName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformMissingProvisionerBasicStr = `
aws_instance.web
  provisioner.shell
provisioner.shell
`

const testTransformCloseProvisionerBasicStr = `
aws_instance.web
  provisioner.shell
provisioner.shell
provisioner.shell (close)
  aws_instance.web
`
