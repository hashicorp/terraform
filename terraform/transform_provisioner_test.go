package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/dag"
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

		var state State
		state.init()
		state.Modules = []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Primary: &InstanceState{ID: "foo"},
					},
				},
			},

			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Primary: &InstanceState{ID: "foo"},
					},
				},
			},
		}

		tf := &StateTransformer{
			Concrete: concreteResource,
			State:    &state,
		}
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
	expected := strings.TrimSpace(testTransformMissingProvisionerModuleStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
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
