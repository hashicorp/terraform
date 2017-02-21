package terraform

import (
	"strings"
	"testing"
)

func TestDiffTransformer_nilDiff(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &DiffTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	if len(g.Vertices()) > 0 {
		t.Fatal("graph should be empty")
	}
}

func TestDiffTransformer(t *testing.T) {
	g := Graph{Path: RootModulePath}
	tf := &DiffTransformer{
		Module: testModule(t, "transform-diff-basic"),
		Diff: &Diff{
			Modules: []*ModuleDiff{
				&ModuleDiff{
					Path: []string{"root"},
					Resources: map[string]*InstanceDiff{
						"aws_instance.foo": &InstanceDiff{
							Attributes: map[string]*ResourceAttrDiff{
								"name": &ResourceAttrDiff{
									Old: "",
									New: "foo",
								},
							},
						},
					},
				},
			},
		},
	}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformDiffBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformDiffBasicStr = `
aws_instance.foo
`
