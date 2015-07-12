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
		t.Fatalf("expected:\n\n%s\n\nbad:\n\n%s", expected, actual)
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

func TestPruneDestroyTransformer_countDec(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	diff := &Diff{}
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar.1": &ResourceState{
						Primary: &InstanceState{},
					},
					"aws_instance.bar.2": &ResourceState{
						Primary: &InstanceState{},
					},
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
		tf := &PruneDestroyTransformer{Diff: diff, State: state}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyCountDecStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer_countState(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	diff := &Diff{}
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Primary: &InstanceState{},
					},
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
		tf := &PruneDestroyTransformer{Diff: diff, State: state}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyCountStateStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer_prefixMatch(t *testing.T) {
	mod := testModule(t, "transform-destroy-prefix")

	diff := &Diff{}
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo-bar.0": &ResourceState{
						Primary: &InstanceState{ID: "foo"},
					},

					"aws_instance.foo-bar.1": &ResourceState{
						Primary: &InstanceState{ID: "foo"},
					},
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
		tf := &PruneDestroyTransformer{Diff: diff, State: state}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyPrefixStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestPruneDestroyTransformer_tainted(t *testing.T) {
	mod := testModule(t, "transform-destroy-basic")

	diff := &Diff{}
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.bar": &ResourceState{
						Tainted: []*InstanceState{
							&InstanceState{ID: "foo"},
						},
					},
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
		tf := &PruneDestroyTransformer{Diff: diff, State: state}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformPruneDestroyTaintedStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformDestroyBasicStr = `
aws_instance.bar
  aws_instance.bar (destroy tainted)
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy tainted)
aws_instance.bar (destroy)
aws_instance.foo
  aws_instance.foo (destroy tainted)
  aws_instance.foo (destroy)
aws_instance.foo (destroy tainted)
  aws_instance.bar (destroy tainted)
aws_instance.foo (destroy)
  aws_instance.bar (destroy)
`

const testTransformPruneDestroyBasicStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
`

const testTransformPruneDestroyBasicDiffStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
`

const testTransformPruneDestroyCountStr = `
aws_instance.bar
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy)
aws_instance.foo
`

const testTransformPruneDestroyCountDecStr = `
aws_instance.bar
  aws_instance.bar (destroy)
  aws_instance.foo
aws_instance.bar (destroy)
aws_instance.foo
`

const testTransformPruneDestroyCountStateStr = `
aws_instance.bar
  aws_instance.foo
aws_instance.foo
`

const testTransformPruneDestroyPrefixStr = `
aws_instance.foo
aws_instance.foo-bar
  aws_instance.foo-bar (destroy)
aws_instance.foo-bar (destroy)
`

const testTransformPruneDestroyTaintedStr = `
aws_instance.bar
  aws_instance.bar (destroy tainted)
  aws_instance.foo
aws_instance.bar (destroy tainted)
aws_instance.foo
`

const testTransformCreateBeforeDestroyBasicStr = `
aws_instance.web
  aws_instance.web (destroy tainted)
aws_instance.web (destroy tainted)
  aws_load_balancer.lb (destroy tainted)
aws_instance.web (destroy)
  aws_instance.web
  aws_load_balancer.lb
  aws_load_balancer.lb (destroy)
aws_load_balancer.lb
  aws_instance.web
  aws_load_balancer.lb (destroy tainted)
  aws_load_balancer.lb (destroy)
aws_load_balancer.lb (destroy tainted)
aws_load_balancer.lb (destroy)
`

const testTransformCreateBeforeDestroyTwiceStr = `
aws_autoscale.bar
  aws_autoscale.bar (destroy tainted)
  aws_lc.foo
aws_autoscale.bar (destroy tainted)
aws_autoscale.bar (destroy)
  aws_autoscale.bar
aws_lc.foo
  aws_lc.foo (destroy tainted)
aws_lc.foo (destroy tainted)
  aws_autoscale.bar (destroy tainted)
aws_lc.foo (destroy)
  aws_autoscale.bar
  aws_autoscale.bar (destroy)
  aws_lc.foo
`
