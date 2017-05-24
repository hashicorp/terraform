package terraform

import (
	"strings"
	"testing"
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

	{
		transform := &AttachResourceConfigTransformer{Module: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{Providers: []string{"aws"}}
		if err := transform.Transform(&g); err != nil {
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

func TestProviderTransformer_moduleChild(t *testing.T) {
	g := Graph{Path: RootModulePath}

	{
		tf := &ImportStateTransformer{
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: "module.moo.foo_instance.qux",
					ID:   "bar",
				},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &ProviderTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformProviderModuleChildStr)
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
		transform := &AttachResourceConfigTransformer{Module: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{Providers: []string{"aws"}}
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
		&MissingProviderTransformer{Providers: []string{"aws"}},
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
	expected := strings.TrimSpace(``)
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
		transform := &AttachResourceConfigTransformer{Module: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &MissingProviderTransformer{Providers: []string{"aws", "foo", "bar"}}
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

func TestMissingProviderTransformer_moduleChild(t *testing.T) {
	g := Graph{Path: RootModulePath}

	// We use the import state transformer since at the time of writing
	// this test it is the first and only transformer that will introduce
	// multiple module-path nodes at a single go.
	{
		tf := &ImportStateTransformer{
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: "module.moo.foo_instance.qux",
					ID:   "bar",
				},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProviderModuleChildStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestMissingProviderTransformer_moduleGrandchild(t *testing.T) {
	g := Graph{Path: RootModulePath}

	// We use the import state transformer since at the time of writing
	// this test it is the first and only transformer that will introduce
	// multiple module-path nodes at a single go.
	{
		tf := &ImportStateTransformer{
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: "module.a.module.b.foo_instance.qux",
					ID:   "bar",
				},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformMissingProviderModuleGrandchildStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestParentProviderTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}

	// Introduce a cihld module
	{
		tf := &ImportStateTransformer{
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: "module.moo.foo_instance.qux",
					ID:   "bar",
				},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Add the missing modules
	{
		tf := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Connect parents
	{
		tf := &ParentProviderTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformParentProviderStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestParentProviderTransformer_moduleGrandchild(t *testing.T) {
	g := Graph{Path: RootModulePath}

	// We use the import state transformer since at the time of writing
	// this test it is the first and only transformer that will introduce
	// multiple module-path nodes at a single go.
	{
		tf := &ImportStateTransformer{
			Targets: []*ImportTarget{
				&ImportTarget{
					Addr: "module.a.module.b.foo_instance.qux",
					ID:   "bar",
				},
			},
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &MissingProviderTransformer{Providers: []string{"foo", "bar"}}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	// Connect parents
	{
		tf := &ParentProviderTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformParentProviderModuleGrandchildStr)
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
		transform := &AttachResourceConfigTransformer{Module: mod}
		if err := transform.Transform(&g); err != nil {
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

const testTransformMissingProviderModuleChildStr = `
module.moo.foo_instance.qux (import id: bar)
module.moo.provider.foo
provider.foo
`

const testTransformMissingProviderModuleGrandchildStr = `
module.a.module.b.foo_instance.qux (import id: bar)
module.a.module.b.provider.foo
module.a.provider.foo
provider.foo
`

const testTransformParentProviderStr = `
module.moo.foo_instance.qux (import id: bar)
module.moo.provider.foo
  provider.foo
provider.foo
`

const testTransformParentProviderModuleGrandchildStr = `
module.a.module.b.foo_instance.qux (import id: bar)
module.a.module.b.provider.foo
  module.a.provider.foo
module.a.provider.foo
  provider.foo
provider.foo
`

const testTransformProviderModuleChildStr = `
module.moo.foo_instance.qux (import id: bar)
  module.moo.provider.foo
module.moo.provider.foo
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
