// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestExcludesTransformer(t *testing.T) {
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
		transform := &ExcludesTransformer{
			Excludes: []addrs.Targetable{
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
aws_instance.notme
aws_subnet.me
  aws_vpc.me
aws_subnet.notme
aws_vpc.me
aws_vpc.notme
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

func TestExcludesTransformer_downstream(t *testing.T) {
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
		transform := &ExcludesTransformer{
			Excludes: []addrs.Targetable{
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
	// Even though we only asked to exclude the grandchild resource, all of the
	// outputs that descend from it are also excluded.
	expected := strings.TrimSpace(`
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
  module.child.aws_instance.foo
output.child_id (expand)
  module.child.output.id (expand)
output.root_id (expand)
  aws_instance.foo
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

// This tests the ExcludesTransformer excluding a whole module,
// rather than a resource within a module instance.
func TestExcludesTransformer_wholeModule(t *testing.T) {
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
		transform := &ExcludesTransformer{
			Excludes: []addrs.Targetable{
				addrs.RootModule.
					Child("child").
					Child("grandchild"),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to exclude the grandchild module, all of the
	// outputs that descend from it are also excluded.
	expected := strings.TrimSpace(`
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
  module.child.aws_instance.foo
output.child_id (expand)
  module.child.output.id (expand)
output.root_id (expand)
  aws_instance.foo
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}
