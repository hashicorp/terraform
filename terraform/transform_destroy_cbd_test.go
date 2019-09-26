package terraform

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
)

func cbdTestGraph(t *testing.T, mod string, changes *plans.Changes) *Graph {
	module := testModule(t, mod)

	applyBuilder := &ApplyGraphBuilder{
		Config:     module,
		Changes:    changes,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}
	g, err := (&BasicGraphBuilder{
		Steps: cbdTestSteps(applyBuilder.Steps()),
		Name:  "ApplyGraphBuilder",
	}).Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return filterInstances(g)
}

// override the apply graph builder to halt the process after CBD
func cbdTestSteps(steps []GraphTransformer) []GraphTransformer {
	found := false
	var i int
	var t GraphTransformer
	for i, t = range steps {
		if _, ok := t.(*CBDEdgeTransformer); ok {
			found = true
			break
		}
	}

	if !found {
		panic("CBDEdgeTransformer not found")
	}

	return steps[:i+1]
}

// remove extra nodes for easier test comparisons
func filterInstances(g *Graph) *Graph {
	for _, v := range g.Vertices() {
		if _, ok := v.(GraphNodeResourceInstance); !ok {
			g.Remove(v)
		}

	}
	return g
}

func TestCBDEdgeTransformer(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	g := cbdTestGraph(t, "transform-destroy-cbd-edge-basic", changes)
	g = filterInstances(g)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
(?m)test_object.A
test_object.A \(destroy deposed \w+\)
  test_object.A
  test_object.B
test_object.B
  test_object.A
`))

	if !expected.MatchString(actual) {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformer_depNonCBDCount(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B[0]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B[1]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	g := cbdTestGraph(t, "transform-cbd-destroy-edge-count", changes)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
(?m)test_object.A
test_object.A \(destroy deposed \w+\)
  test_object.A
  test_object.B\[0\]
  test_object.B\[1\]
test_object.B\[0\]
  test_object.A
test_object.B\[1\]
  test_object.A`))

	if !expected.MatchString(actual) {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformer_depNonCBDCountBoth(t *testing.T) {
	changes := &plans.Changes{
		Resources: []*plans.ResourceInstanceChangeSrc{
			{
				Addr: mustResourceInstanceAddr("test_object.A[0]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.A[1]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B[0]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.B[1]"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	g := cbdTestGraph(t, "transform-cbd-destroy-edge-both-count", changes)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
test_object.A \(destroy deposed \w+\)
  test_object.A\[0\]
  test_object.A\[1\]
  test_object.B\[0\]
  test_object.B\[1\]
test_object.A \(destroy deposed \w+\)
  test_object.A\[0\]
  test_object.A\[1\]
  test_object.B\[0\]
  test_object.B\[1\]
test_object.A\[0\]
test_object.A\[1\]
test_object.B\[0\]
  test_object.A\[0\]
  test_object.A\[1\]
test_object.B\[1\]
  test_object.A\[0\]
  test_object.A\[1\]
`))

	if !expected.MatchString(actual) {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}
