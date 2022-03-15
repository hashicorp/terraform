package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestRootTransformer(t *testing.T) {
	mod := testModule(t, "transform-root-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &RootTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRootBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}

	root, err := g.Root()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if _, ok := root.(graphNodeRoot); !ok {
		t.Fatalf("bad: %#v", root)
	}
}

const testTransformRootBasicStr = `
aws_instance.foo
  provider["registry.terraform.io/hashicorp/aws"]
do_droplet.bar
  provider["registry.terraform.io/hashicorp/do"]
provider["registry.terraform.io/hashicorp/aws"]
provider["registry.terraform.io/hashicorp/do"]
root
  aws_instance.foo
  do_droplet.bar
`
