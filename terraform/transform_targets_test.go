package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestTargetsTransformer(t *testing.T) {
	mod := testModule(t, "transform-targets-basic")

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
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_instance", "me",
				),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.me
  aws_subnet.me
aws_subnet.me
  aws_vpc.me
aws_vpc.me
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

func TestTargetsTransformer_downstream(t *testing.T) {
	mod := testModule(t, "transform-targets-downstream")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		transform := &ConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &OutputTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.
					Child("child", addrs.NoKey).
					Child("grandchild", addrs.NoKey).
					Resource(
						addrs.ManagedResourceMode, "aws_instance", "foo",
					),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to target the grandchild resource, all of the
	// outputs that descend from it are also targeted.
	expected := strings.TrimSpace(`
module.child.module.grandchild.aws_instance.foo
module.child.module.grandchild.output.id
  module.child.module.grandchild.aws_instance.foo
module.child.output.grandchild_id
  module.child.module.grandchild.output.id
output.grandchild_id
  module.child.output.grandchild_id
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

func TestTargetsTransformer_destroy(t *testing.T) {
	mod := testModule(t, "transform-targets-destroy")

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
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_instance", "me",
				),
			},
			Destroy: true,
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_elb.me
  aws_instance.me
aws_instance.me
aws_instance.metoo
  aws_instance.me
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}
