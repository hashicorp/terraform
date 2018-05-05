package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
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
		transform := &MissingProviderTransformer{
			Providers: []string{"aws", "do"},
		}
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
		t.Fatalf("bad:\n\n%s", actual)
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
  provider.aws
do_droplet.bar
  provider.do
provider.aws
provider.do
root
  aws_instance.foo
  do_droplet.bar
`
