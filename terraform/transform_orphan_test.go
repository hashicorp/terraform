package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestOrphanTransformer(t *testing.T) {
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_modules(t *testing.T) {
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

			// Orphan module
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanModulesStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_modulesDeps(t *testing.T) {
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

			// Orphan module
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
				Dependencies: []string{
					"aws_instance.foo",
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanModulesDepsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_modulesDepsOrphan(t *testing.T) {
	mod := testModule(t, "transform-orphan-modules")
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
				},
			},

			// Orphan module
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
				Dependencies: []string{
					"aws_instance.web",
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanModulesDepsOrphanStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_modulesNoRoot(t *testing.T) {
	mod := testModule(t, "transform-orphan-modules")
	state := &State{
		Modules: []*ModuleState{
			// Orphan module
			&ModuleState{
				Path: []string{RootModuleName, "foo"},
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanModulesNoRootStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_resourceDepends(t *testing.T) {
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
						Dependencies: []string{
							"aws_instance.web",
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

	transform := &OrphanTransformer{State: state, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceDependsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanTransformer_nilState(t *testing.T) {
	mod := testModule(t, "transform-orphan-basic")

	g := Graph{Path: RootModulePath}
	{
		tf := &ConfigTransformer{Module: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	transform := &OrphanTransformer{State: nil, Module: mod}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanNilStateStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestGraphNodeOrphanModule_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeOrphanModule)
	var _ dag.NamedVertex = new(graphNodeOrphanModule)
	var _ GraphNodeExpandable = new(graphNodeOrphanModule)
}

func TestGraphNodeOrphanResource_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeOrphanResource)
	var _ dag.NamedVertex = new(graphNodeOrphanResource)
	var _ GraphNodeProviderConsumer = new(graphNodeOrphanResource)
}

func TestGraphNodeOrphanResource_ProvidedBy(t *testing.T) {
	n := &graphNodeOrphanResource{ResourceName: "aws_instance.foo"}
	if v := n.ProvidedBy(); v[0] != "aws" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeOrphanResource_ProvidedBy_alias(t *testing.T) {
	n := &graphNodeOrphanResource{ResourceName: "aws_instance.foo", Provider: "aws.bar"}
	if v := n.ProvidedBy(); v[0] != "aws.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformOrphanBasicStr = `
aws_instance.db (orphan)
aws_instance.web
`

const testTransformOrphanModulesStr = `
aws_instance.foo
module.foo (orphan)
`

const testTransformOrphanModulesDepsStr = `
aws_instance.foo
module.foo (orphan)
  aws_instance.foo
`

const testTransformOrphanModulesDepsOrphanStr = `
aws_instance.foo
aws_instance.web (orphan)
module.foo (orphan)
  aws_instance.web (orphan)
`

const testTransformOrphanNilStateStr = `
aws_instance.web
`

const testTransformOrphanResourceDependsStr = `
aws_instance.db (orphan)
  aws_instance.web
aws_instance.web
`

const testTransformOrphanModulesNoRootStr = `
aws_instance.foo
module.foo (orphan)
`
