package terraform

import (
	"strings"
	"testing"
)

func TestAddOutputOrphanTransformer(t *testing.T) {
	mod := testModule(t, "transform-orphan-output-basic")
	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: RootModulePath,
				Outputs: map[string]*OutputState{
					"foo": &OutputState{
						Value: "bar",
						Type:  "string",
					},
					"bar": &OutputState{
						Value: "baz",
						Type:  "string",
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

	transform := &AddOutputOrphanTransformer{State: state}
	if err := transform.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanOutputBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformOrphanOutputBasicStr = `
output.bar (orphan)
output.foo
`
