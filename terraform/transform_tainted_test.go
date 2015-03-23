package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
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

func TestGraphNodeTaintedResource_impl(t *testing.T) {
	var _ dag.Vertex = new(graphNodeTaintedResource)
	var _ dag.NamedVertex = new(graphNodeTaintedResource)
	var _ GraphNodeProviderConsumer = new(graphNodeTaintedResource)
}

func TestGraphNodeTaintedResource_ProvidedBy(t *testing.T) {
	n := &graphNodeTaintedResource{ResourceName: "aws_instance.foo"}
	if v := n.ProvidedBy(); v[0] != "aws" {
		t.Fatalf("bad: %#v", v)
	}
}

func TestGraphNodeTaintedResource_ProvidedBy_alias(t *testing.T) {
	n := &graphNodeTaintedResource{ResourceName: "aws_instance.foo", Provider: "aws.bar"}
	if v := n.ProvidedBy(); v[0] != "aws.bar" {
		t.Fatalf("bad: %#v", v)
	}
}

const testTransformTaintedBasicStr = `
aws_instance.web
aws_instance.web (tainted #1)
`
