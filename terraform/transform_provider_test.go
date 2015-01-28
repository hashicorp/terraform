package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	transform := &ProviderTransformer{}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestMissingProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	transform := &MissingProviderTransformer{Providers: []string{"foo"}}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeMissingProvider_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeMissingProvider)
	var _ dag.NamedVertex = new(graphNodeMissingProvider)
	var _ GraphNodeProvider = new(graphNodeMissingProvider)
}

func TestGraphNodeMissingProvider_ProviderName(t *testing.T) {
	n := &graphNodeMissingProvider{ProviderNameValue: "foo"}
	if v := n.ProviderName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformProviderBasicStr = `
aws_instance.web
  provider.aws
provider.aws
`

const testTransformMissingProviderBasicStr = `
aws_instance.web
provider.aws
provider.foo
`
