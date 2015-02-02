package terraform

import (
	"strings"
	"testing"
)

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

const testBuiltinGraphBuilderBasicStr = `
aws_instance.db
  provider.aws
aws_instance.web
  aws_instance.db
  provider.aws
provider.aws
`
