package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/dag"
)

func TestBasicGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(BasicGraphBuilder)
}

func TestBasicGraphBuilder(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBasicGraphBuilderStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestBasicGraphBuilder_validate(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
			&testBasicGraphBuilderTransform{2},
		},
		Validate: true,
	}

	_, err := b.Build(addrs.RootModuleInstance)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestBasicGraphBuilder_validateOff(t *testing.T) {
	b := &BasicGraphBuilder{
		Steps: []GraphTransformer{
			&testBasicGraphBuilderTransform{1},
			&testBasicGraphBuilderTransform{2},
		},
		Validate: false,
	}

	_, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}

type testBasicGraphBuilderTransform struct {
	V dag.Vertex
}

func (t *testBasicGraphBuilderTransform) Transform(g *Graph) error {
	g.Add(t.V)
	return nil
}

const testBasicGraphBuilderStr = `
1
`
