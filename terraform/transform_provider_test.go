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

func TestCloseProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
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
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCloseProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCloseProviderTransformer_withTargets(t *testing.T) {
	mod := testModule(t, "transform-provider-basic")

	g := Graph{Path: RootModulePath}
	transforms := []GraphTransformer{
		&ConfigTransformer{Module: mod},
		&ProviderTransformer{},
		&CloseProviderTransformer{},
		&TargetsTransformer{
			Targets: []string{"something.else"},
		},
	}

	for _, tr := range transforms {
		if err := tr.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
provider.aws
provider.aws (close)
  provider.aws
	`)
	if actual != expected {
		t.Fatalf("expected:%s\n\ngot:\n\n%s", expected, actual)
	}
}

func TestMissingProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-missing")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-prune")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{Providers: []string{"foo"}}
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
		transform := &CloseProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &PruneProviderTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneProviderBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestDisableProviderTransformer(t *testing.T) {
	mod := testModule(t, "transform-provider-disable")

	g := Graph{Path: RootModulePath}
	transforms := []GraphTransformer{
		&ConfigTransformer{Module: mod},
		&MissingProviderTransformer{Providers: []string{"aws"}},
		&ProviderTransformer{},
		&DisableProviderTransformer{},
		&CloseProviderTransformer{},
		&PruneProviderTransformer{},
	}

	for _, tr := range transforms {
		if err := tr.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDisableProviderBasicStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

func TestDisableProviderTransformer_keep(t *testing.T) {
	mod := testModule(t, "transform-provider-disable-keep")

	g := Graph{Path: RootModulePath}
	transforms := []GraphTransformer{
		&ConfigTransformer{Module: mod},
		&MissingProviderTransformer{Providers: []string{"aws"}},
		&ProviderTransformer{},
		&DisableProviderTransformer{},
		&CloseProviderTransformer{},
		&PruneProviderTransformer{},
	}

	for _, tr := range transforms {
		if err := tr.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDisableProviderKeepStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

func TestGraphNodeProvider_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeProvider)
	var _ dag.NamedVertex = new(graphNodeProvider)
	var _ GraphNodeProvider = new(graphNodeProvider)
}

func TestGraphNodeProvider_ProviderName(t *testing.T) {
	n := &graphNodeProvider{ProviderNameValue: "foo"}
	if v := n.ProviderName(); v != "foo" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformProviderBasicStr = `
aws_instance.web
  provider.aws
provider.aws
`

const testTransformCloseProviderBasicStr = `
aws_instance.web
  provider.aws
provider.aws
provider.aws (close)
  aws_instance.web
  provider.aws
`

const testTransformMissingProviderBasicStr = `
aws_instance.web
foo_instance.web
provider.aws
provider.aws (close)
  aws_instance.web
  provider.aws
provider.foo
provider.foo (close)
  foo_instance.web
  provider.foo
`

const testTransformPruneProviderBasicStr = `
foo_instance.web
  provider.foo
provider.foo
provider.foo (close)
  foo_instance.web
  provider.foo
`

const testTransformDisableProviderBasicStr = `
module.child
  provider.aws (disabled)
  var.foo
provider.aws (close)
  module.child
  provider.aws (disabled)
provider.aws (disabled)
var.foo
`

const testTransformDisableProviderKeepStr = `
aws_instance.foo
  provider.aws
module.child
  provider.aws
  var.foo
provider.aws
provider.aws (close)
  aws_instance.foo
  module.child
  provider.aws
var.foo
`
