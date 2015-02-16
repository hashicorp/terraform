package terraform

import (
	"strings"
	"testing"
)

func TestDestroyTransformer(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestDestroyTransformer_deps(t *testing.T) {
	mod := testModule(t, "transform-destroy-deps")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDestroyDepsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCreateBeforeDestroyTransformer(t *testing.T) {
	mod := testModule(t, "transform-create-before-destroy-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CreateBeforeDestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCreateBeforeDestroyBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestCreateBeforeDestroyTransformer_twice(t *testing.T) {
	mod := testModule(t, "transform-create-before-destroy-twice")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &CreateBeforeDestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformCreateBeforeDestroyTwiceStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer(t *testing.T) {
	var diff *Diff
	mod := testModule(t, "transform-destroy-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &PruneDestroyTransformer{Diff: diff}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer_diff(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: RootModulePath,
				Resources: map[string]*InstanceDiff{
					"aws_instance.bar": &InstanceDiff{},
				},
			},
		},
	}

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &PruneDestroyTransformer{Diff: diff}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyBasicDiffStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer_count(t *testing.T) {
	mod := testModule(t, "transform-destroy-prune-count")

	diff := &Diff{}

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &DestroyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &PruneDestroyTransformer{Diff: diff}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyCountStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformDestroyBasicStr = `
aws_instance.bar
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy)
aws_instance.foo
  aws_instance.foo (destroy)
aws_instance.foo (destroy)
  aws_instance.bar (destroy)
`

const testTransformDestroyDepsStr = `
aws_asg.bar
  aws_asg.bar (destroy)
  aws_lc.foo
aws_asg.bar (destroy)
aws_lc.foo
  aws_lc.foo (destroy)
aws_lc.foo (destroy)
  aws_asg.bar (destroy)
`

const testTransformPruneDestroyBasicStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
`

const testTransformPruneDestroyBasicDiffStr = `
aws_instance.bar
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy)
aws_instance.foo
`

const testTransformPruneDestroyCountStr = `
aws_instance.bar
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy)
aws_instance.foo
`

const testTransformCreateBeforeDestroyBasicStr = `
aws_instance.web
aws_instance.web (destroy)
  aws_instance.web
  aws_load_balancer.lb
  aws_load_balancer.lb (destroy)
aws_load_balancer.lb
  aws_instance.web
  aws_load_balancer.lb (destroy)
aws_load_balancer.lb (destroy)
`

const testTransformCreateBeforeDestroyTwiceStr = `
aws_autoscale.bar
  aws_lc.foo
aws_autoscale.bar (destroy)
  aws_autoscale.bar
aws_lc.foo
aws_lc.foo (destroy)
  aws_autoscale.bar
  aws_autoscale.bar (destroy)
  aws_lc.foo
`
