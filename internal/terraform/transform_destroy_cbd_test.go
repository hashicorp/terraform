// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
)

func cbdTestGraph(t *testing.T, mod string, changes *plans.Changes, state *states.State) *Graph {
	module := testModule(t, mod)

	applyBuilder := &ApplyGraphBuilder{
		Config:  module,
		Changes: changes,
		Plugins: simpleMockPluginLibrary(),
		State:   state,
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

	// re-add the root node so we have a valid graph for a walk, then reduce
	// the graph for less output
	steps = append(steps[:i+1], &CloseRootModuleTransformer{})
	steps = append(steps, &TransitiveReductionTransformer{})

	return steps
}

// remove extra nodes for easier test comparisons
func filterInstances(g *Graph) *Graph {
	for _, v := range g.Vertices() {
		if _, ok := v.(GraphNodeResourceInstance); !ok {
			// connect around the node to remove it without breaking deps
			for _, down := range g.DownEdges(v) {
				for _, up := range g.UpEdges(v) {
					g.Connect(dag.BasicEdge(up, down))
				}
			}

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

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	g := cbdTestGraph(t, "transform-destroy-cbd-edge-basic", changes, state)
	g = filterInstances(g)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
(?m)test_object.A
test_object.A \(destroy deposed \w+\)
  test_object.B
test_object.B
  test_object.A
`))

	if !expected.MatchString(actual) {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestCBDEdgeTransformerMulti(t *testing.T) {
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
					Action: plans.CreateThenDelete,
				},
			},
			{
				Addr: mustResourceInstanceAddr("test_object.C"),
				ChangeSrc: plans.ChangeSrc{
					Action: plans.Update,
				},
			},
		},
	}

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"B"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.C").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"C","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{
				mustConfigResourceAddr("test_object.A"),
				mustConfigResourceAddr("test_object.B"),
			},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	g := cbdTestGraph(t, "transform-destroy-cbd-edge-multi", changes, state)
	g = filterInstances(g)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
(?m)test_object.A
test_object.A \(destroy deposed \w+\)
  test_object.C
test_object.B
test_object.B \(destroy deposed \w+\)
  test_object.C
test_object.C
  test_object.A
  test_object.B
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

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	g := cbdTestGraph(t, "transform-cbd-destroy-edge-count", changes, state)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
(?m)test_object.A
test_object.A \(destroy deposed \w+\)
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

	state := states.NewState()
	root := state.EnsureModule(addrs.RootModuleInstance)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.A[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"A"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("test_object.B[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:       states.ObjectReady,
			AttrsJSON:    []byte(`{"id":"B","test_list":["x"]}`),
			Dependencies: []addrs.ConfigResource{mustConfigResourceAddr("test_object.A")},
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/test"]`),
	)

	g := cbdTestGraph(t, "transform-cbd-destroy-edge-both-count", changes, state)

	actual := strings.TrimSpace(g.String())
	expected := regexp.MustCompile(strings.TrimSpace(`
test_object.A\[0\]
test_object.A\[0\] \(destroy deposed \w+\)
  test_object.B\[0\]
  test_object.B\[1\]
test_object.A\[1\]
test_object.A\[1\] \(destroy deposed \w+\)
  test_object.B\[0\]
  test_object.B\[1\]
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
