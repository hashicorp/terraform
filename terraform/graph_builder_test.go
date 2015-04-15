package terraform

import (
	"reflect"
	"strings"
	"testing"

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

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
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
	}

	_, err := b.Build(RootModulePath)
	if err == nil {
		t.Fatal("should error")
	}
}

func TestBuiltinGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(BuiltinGraphBuilder)
}

// This test is not meant to test all the transforms but rather just
// to verify we get some basic sane graph out. Special tests to ensure
// specific ordering of steps should be added in other tests.
func TestBuiltinGraphBuilder(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root: testModule(t, "graph-builder-basic"),
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

// This tests a cycle we got when a CBD resource depends on a non-CBD
// resource. This cycle shouldn't happen in the general case anymore.
func TestBuiltinGraphBuilder_cbdDepNonCbd(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root: testModule(t, "graph-builder-cbd-non-cbd"),
	}

	_, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

/*
TODO: This exposes a really bad bug we need to fix after we merge
the f-ast-branch. This bug still exists in master.

// This test tests that the graph builder properly expands modules.
func TestBuiltinGraphBuilder_modules(t *testing.T) {
	b := &BuiltinGraphBuilder{
		Root: testModule(t, "graph-builder-modules"),
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testBuiltinGraphBuilderModuleStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}
*/

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

const testBuiltinGraphBuilderBasicStr = `
aws_instance.db
  provider.aws
aws_instance.web
  aws_instance.db
provider.aws
`

const testBuiltinGraphBuilderModuleStr = `
aws_instance.web
  aws_instance.web (destroy)
aws_instance.web (destroy)
  aws_security_group.firewall
  module.consul (expanded)
  provider.aws
aws_security_group.firewall
  aws_security_group.firewall (destroy)
aws_security_group.firewall (destroy)
  provider.aws
module.consul (expanded)
  aws_security_group.firewall
  provider.aws
provider.aws
`
