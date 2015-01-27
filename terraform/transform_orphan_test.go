package terraform

import (
	"strings"
	"testing"
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

	transform := &OrphanTransformer{State: state, Config: mod.Config()}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanBasicStr)
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

	transform := &OrphanTransformer{State: state, Config: mod.Config()}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceDependsStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformOrphanBasicStr = `
aws_instance.db (orphan)
aws_instance.web
`

const testTransformOrphanResourceDependsStr = `
aws_instance.db (orphan)
  aws_instance.web
aws_instance.web
`
