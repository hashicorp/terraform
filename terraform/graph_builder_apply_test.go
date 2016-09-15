package terraform

import (
	"reflect"
	"strings"
	"testing"
)

func TestApplyGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(ApplyGraphBuilder)
}

func TestApplyGraphBuilder(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					// Verify noop doesn't show up in graph
					"aws_instance.noop": &InstanceDiff{},

					"aws_instance.create": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},

			&ModuleDiff{
				Path: []string{"root", "child"},
				Resources: map[string]*InstanceDiff{
					"aws_instance.create": &InstanceDiff{
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
	}

	b := &ApplyGraphBuilder{
		Module:       testModule(t, "graph-builder-apply-basic"),
		Diff:         diff,
		Providers:    []string{"aws"},
		Provisioners: []string{"exec"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

const testApplyGraphBuilderStr = `
aws_instance.create
  provider.aws
module.child.aws_instance.create
  module.child.provider.aws
  provisioner.exec
module.child.provider.aws
  provider.aws
provider.aws
provisioner.exec
root
  aws_instance.create
  module.child.aws_instance.create
`
