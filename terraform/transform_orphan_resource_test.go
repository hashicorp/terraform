package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestOrphanResourceTransformer(t *testing.T) {
	mod := testModule(t, "transform-orphan-basic")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					// The orphan
					"aws_instance.db": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state, Module: mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_nilModule(t *testing.T) {
	mod := testModule(t, "transform-orphan-basic")
	state := &State{
		Modules: []*ModuleState{nil},
	}

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state, Module: mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
}

func TestOrphanResourceTransformer_countGood(t *testing.T) {
	mod := testModule(t, "transform-orphan-count")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state, Module: mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_countBad(t *testing.T) {
	mod := testModule(t, "transform-orphan-count-empty")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state, Module: mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountBadStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_modules(t *testing.T) {
	mod := testModule(t, "transform-orphan-modules")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},

			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
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
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state, Module: mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformOrphanResourceBasicStr = `
aws_instance.db (orphan)
aws_instance.web
`

const testTransformOrphanResourceCountStr = `
aws_instance.foo
`

const testTransformOrphanResourceCountBadStr = `
aws_instance.foo[0] (orphan)
aws_instance.foo[1] (orphan)
`

const testTransformOrphanResourceModulesStr = `
aws_instance.foo
module.child.aws_instance.web (orphan)
`

func testOrphanResourceConcreteFunc(a *NodeAbstractResource) dag.Vertex {
	return &testOrphanResourceConcrete{a}
}

type testOrphanResourceConcrete struct {
	*NodeAbstractResource
}

func (n *testOrphanResourceConcrete) Name() string {
	return fmt.Sprintf("%s (orphan)", n.NodeAbstractResource.Name())
}
