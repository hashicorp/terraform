package terraform

import (
	"strings"
	"testing"
)

func TestTaintedTransformer(t *testing.T) {
	mod := testModule(t, "transform-tainted-basic")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
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

	transform := &TaintedTransformer{State: state}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformTaintedBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformTaintedBasicStr = `
aws_instance.web
aws_instance.web (tainted #1)
`
