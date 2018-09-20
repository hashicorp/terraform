package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
)

func TestMissingProvisionerTransformer(t *testing.T) {
	mod := testModule(t, "transform-provisioner-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
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

func TestMissingProvisionerTransformer_module(t *testing.T) {
	mod := testModule(t, "transform-provisioner-module")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		concreteResource := func(a *NodeAbstractResourceInstance) dag.Vertex {
			return a
		}

		state := states.BuildState(func(s *states.SyncState) {
			s.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				&states.ResourceInstanceObjectSrc{
					AttrsFlat: map[string]string{
						"id": "foo",
					},
					Status: states.ObjectReady,
				},
				addrs.ProviderConfig{
					Type: "aws",
				}.Absolute(addrs.RootModuleInstance),
			)
			s.SetResourceInstanceCurrent(
				addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: "aws_instance",
					Name: "foo",
				}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
				&states.ResourceInstanceObjectSrc{
					AttrsFlat: map[string]string{
						"id": "foo",
					},
					Status: states.ObjectReady,
				},
				addrs.ProviderConfig{
					Type: "aws",
				}.Absolute(addrs.RootModuleInstance),
			)
		})

		tf := &StateTransformer{
			ConcreteCurrent: concreteResource,
			State:           state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after StateTransformer:\n%s", g.StringWithNodeTypes())
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProvisionerTransformer{Provisioners: []string{"shell"}}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after MissingProvisionerTransformer:\n%s", g.StringWithNodeTypes())
	}

	{
		transform := &ProvisionerTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
		t.Logf("graph after ProvisionerTransformer:\n%s", g.StringWithNodeTypes())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProvisionerModuleStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCloseProvisionerTransformer(t *testing.T) {
	mod := testModule(t, "transform-provisioner-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
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

const testTransformMissingProvisionerBasicStr = `
aws_instance.web
  provisioner.shell
provisioner.shell
`

const testTransformMissingProvisionerModuleStr = `
aws_instance.foo
  provisioner.shell
module.child.aws_instance.foo
  module.child.provisioner.shell
module.child.provisioner.shell
provisioner.shell
`

const testTransformCloseProvisionerBasicStr = `
aws_instance.web
  provisioner.shell
provisioner.shell
provisioner.shell (close)
  aws_instance.web
`
