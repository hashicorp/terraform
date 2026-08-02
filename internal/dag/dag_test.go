// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// vertex type for our test graph integer nodes
type testV int

func (i testV) Name() string {
	return fmt.Sprint(i)
}

// vertex type for test graph string nodes
type testNamedString string

func (s testNamedString) Name() string {
	return string(s)
}

func TestAcyclicGraphRoot(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(3), testV(1))

	if root, err := g.Root(); err != nil {
		t.Fatalf("err: %s", err)
	} else if root != testV(3) {
		t.Fatalf("bad: %#v", root)
	}
}

func TestAcyclicGraphRoot_cycle(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))
	g.Connect(testV(3), testV(1))

	if _, err := g.Root(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphRoot_multiple(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(3), testV(2))

	if _, err := g.Root(); err == nil {
		t.Fatal("should error")
	}
}

func TestAyclicGraphTransReduction(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(1), testV(3))
	g.Connect(testV(2), testV(3))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestAyclicGraphTransReduction_more(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(4))
	g.Connect(testV(2), testV(3))
	g.Connect(testV(2), testV(4))
	g.Connect(testV(3), testV(4))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionMoreStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestAyclicGraphTransReduction_multipleRoots(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(4))
	g.Connect(testV(2), testV(3))
	g.Connect(testV(2), testV(4))
	g.Connect(testV(3), testV(4))

	g.Add(testV(5))
	g.Add(testV(6))
	g.Add(testV(7))
	g.Add(testV(8))
	g.Connect(testV(5), testV(6))
	g.Connect(testV(5), testV(7))
	g.Connect(testV(5), testV(8))
	g.Connect(testV(6), testV(7))
	g.Connect(testV(6), testV(8))
	g.Connect(testV(7), testV(8))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionMultipleRootsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

// use this to simulate slow sort operations
type counter struct {
	name  string
	Calls int64
}

func (s *counter) Name() string {
	s.Calls++
	return s.name
}

func (s *counter) String() string {
	return s.Name()
}

// Make sure we can reduce a sizable, fully-connected graph.
func TestAyclicGraphTransReduction_fullyConnected(t *testing.T) {
	var g AcyclicGraph

	const nodeCount = 200
	nodes := make([]*counter, nodeCount)
	for i := range nodeCount {
		nodes[i] = &counter{name: strconv.Itoa(i)}
	}

	// Add them all to the graph
	for _, n := range nodes {
		g.Add(n)
	}

	// connect them all
	for i := range nodes {
		for j := range nodes {
			if i == j {
				continue
			}
			g.Connect(nodes[i], nodes[j])
		}
	}

	g.TransitiveReduction()

	vertexNameCalls := int64(0)
	for _, n := range nodes {
		vertexNameCalls += n.Calls
	}

	switch {
	case vertexNameCalls > 2*nodeCount:
		// Make calling it more the 2x per node fatal.
		// If we were sorting this would give us roughly ln(n)(n^3) calls, or
		// >59000000 calls for 200 vertices.
		t.Fatalf("VertexName called %d times", vertexNameCalls)
	case vertexNameCalls > 0:
		// we don't expect any calls, but a change here isn't necessarily fatal
		t.Logf("WARNING: VertexName called %d times", vertexNameCalls)
	}
}

func TestAcyclicGraphValidate(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(3), testV(1))

	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestAcyclicGraphValidate_cycle(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(3), testV(1))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(1))

	if err := g.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphValidate_cycleSelf(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Connect(testV(1), testV(1))

	if err := g.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphAncestors(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Add(testV(5))
	g.Connect(testV(0), testV(1))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))
	g.Connect(testV(3), testV(4))
	g.Connect(testV(4), testV(5))

	actual := g.Ancestors(testV(2))

	expected := []Vertex{testV(3), testV(4), testV(5)}

	if actual.Len() != len(expected) {
		t.Fatalf("bad length! expected %#v to have len %d", actual, len(expected))
	}

	for _, e := range expected {
		if !actual.Include(e) {
			t.Fatalf("expected: %#v to include: %#v", expected, actual)
		}
	}
}

func TestAcyclicGraphDescendants(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Add(testV(5))
	g.Connect(testV(0), testV(1))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(3))
	g.Connect(testV(3), testV(4))
	g.Connect(testV(4), testV(5))

	actual := g.Descendants(testV(2))

	expected := []Vertex{testV(0), testV(1)}

	if actual.Len() != len(expected) {
		t.Fatalf("bad length! expected %#v to have len %d", actual, len(expected))
	}

	for _, e := range expected {
		if !actual.Include(e) {
			t.Fatalf("expected: %#v to include: %#v", expected, actual)
		}
	}
}

func TestAcyclicGraphFindDescendants(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(0))
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Add(testV(5))
	g.Add(testV(6))
	g.Connect(testV(0), testV(1))
	g.Connect(testV(1), testV(2))
	g.Connect(testV(2), testV(6))
	g.Connect(testV(3), testV(4))
	g.Connect(testV(4), testV(5))
	g.Connect(testV(5), testV(6))

	actual := g.FirstDescendantsWith(testV(6), func(v Vertex) bool {
		// looking for first odd descendants
		return v.(testV)%2 != 0
	})

	expected := make(Set)
	expected.Add(testV(1))
	expected.Add(testV(5))

	if expected.Intersection(actual).Len() != expected.Len() {
		t.Fatalf("expected %#v, got %#v\n", expected, actual)
	}

	foundOne := g.MatchDescendant(testV(6), func(v Vertex) bool {
		return v.(testV) == 1
	})
	if !foundOne {
		t.Fatal("did not match 1 in the graph")
	}

	foundSix := g.MatchDescendant(testV(6), func(v Vertex) bool {
		return v.(testV) == 6
	})
	if foundSix {
		t.Fatal("6 should not be a descendant of itself")
	}

	foundTen := g.MatchDescendant(testV(6), func(v Vertex) bool {
		return v.(testV) == 10
	})
	if foundTen {
		t.Fatal("10 is not in the graph at all")
	}
}

func TestAcyclicGraphFindAncestors(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(0))
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Add(testV(5))
	g.Add(testV(6))
	g.Connect(testV(1), testV(0))
	g.Connect(testV(2), testV(1))
	g.Connect(testV(6), testV(2))
	g.Connect(testV(4), testV(3))
	g.Connect(testV(5), testV(4))
	g.Connect(testV(6), testV(5))

	actual := g.FirstAncestorsWith(testV(6), func(v Vertex) bool {
		// looking for first odd ancestors
		return v.(testV)%2 != 0
	})

	expected := make(Set)
	expected.Add(testV(1))
	expected.Add(testV(5))

	if expected.Intersection(actual).Len() != expected.Len() {
		t.Fatalf("expected %#v, got %#v\n", expected, actual)
	}

	foundOne := g.MatchAncestor(testV(6), func(v Vertex) bool {
		return v.(testV) == 1
	})
	if !foundOne {
		t.Fatal("did not match 1 in the graph")
	}

	foundSix := g.MatchAncestor(testV(6), func(v Vertex) bool {
		return v.(testV) == 6
	})
	if foundSix {
		t.Fatal("6 should not be a descendant of itself")
	}

	foundTen := g.MatchAncestor(testV(6), func(v Vertex) bool {
		return v.(testV) == 10
	})
	if foundTen {
		t.Fatal("10 is not in the graph at all")
	}
}

func TestAcyclicGraphWalk(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(3), testV(1))

	var visits []Vertex
	var lock sync.Mutex
	err := g.Walk(func(v Vertex) tfdiags.Diagnostics {
		lock.Lock()
		defer lock.Unlock()
		visits = append(visits, v)
		return nil
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := [][]Vertex{
		{testV(1), testV(2), testV(3)},
		{testV(2), testV(1), testV(3)},
	}
	for _, e := range expected {
		if reflect.DeepEqual(visits, e) {
			return
		}
	}

	t.Fatalf("bad: %#v", visits)
}

func TestAcyclicGraphWalk_error(t *testing.T) {
	var g AcyclicGraph
	g.Add(testV(1))
	g.Add(testV(2))
	g.Add(testV(3))
	g.Add(testV(4))
	g.Connect(testV(4), testV(3))
	g.Connect(testV(3), testV(2))
	g.Connect(testV(2), testV(1))

	var visits []Vertex
	var lock sync.Mutex
	err := g.Walk(func(v Vertex) tfdiags.Diagnostics {
		lock.Lock()
		defer lock.Unlock()

		var diags tfdiags.Diagnostics

		if v == testV(2) {
			diags = diags.Append(fmt.Errorf("error"))
			return diags
		}

		visits = append(visits, v)
		return diags
	})
	if err == nil {
		t.Fatal("should error")
	}

	expected := []Vertex{testV(1)}
	if !reflect.DeepEqual(visits, expected) {
		t.Errorf("wrong visits\ngot:  %#v\nwant: %#v", visits, expected)
	}

}

func BenchmarkDAG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		count := 150
		b.StopTimer()
		g := &AcyclicGraph{}

		// create 4 layers of fully connected nodes
		// layer A
		for i := range count {
			g.Add(testNamedString(fmt.Sprintf("A%d", i)))
		}

		// layer B
		for i := range count {
			B := testNamedString(fmt.Sprintf("B%d", i))
			g.Add(B)
			for j := range count {
				g.Connect(B, testNamedString(fmt.Sprintf("A%d", j)))
			}
		}

		// layer C
		for i := range count {
			c := testNamedString(fmt.Sprintf("C%d", i))
			g.Add(c)
			for j := range count {
				// connect them to previous layers so we have something that requires reduction
				g.Connect(c, testNamedString(fmt.Sprintf("A%d", j)))
				g.Connect(c, testNamedString(fmt.Sprintf("B%d", j)))
			}
		}

		// layer D
		for i := range count {
			d := testNamedString(fmt.Sprintf("D%d", i))
			g.Add(d)
			for j := range count {
				g.Connect(d, testNamedString(fmt.Sprintf("A%d", j)))
				g.Connect(d, testNamedString(fmt.Sprintf("B%d", j)))
				g.Connect(d, testNamedString(fmt.Sprintf("C%d", j)))
			}
		}

		b.StartTimer()
		// Find dependencies for every node
		for _, v := range g.Vertices() {
			_ = g.Ancestors(v)
		}

		// reduce the final graph
		g.TransitiveReduction()
	}
}

func TestAcyclicGraphWalkOrder(t *testing.T) {
	/* Sample dependency graph,
	   all edges pointing downwards.
	       1    2
	      / \  /  \
	     3    4    5
	    /      \  /
	   6         7
	           / | \
	          8  9  10
	           \ | /
	             11
	*/

	var g AcyclicGraph
	for i := 1; i <= 11; i++ {
		g.Add(testV(i))
	}
	g.Connect(testV(1), testV(3))
	g.Connect(testV(1), testV(4))
	g.Connect(testV(2), testV(4))
	g.Connect(testV(2), testV(5))
	g.Connect(testV(3), testV(6))
	g.Connect(testV(4), testV(7))
	g.Connect(testV(5), testV(7))
	g.Connect(testV(7), testV(8))
	g.Connect(testV(7), testV(9))
	g.Connect(testV(7), testV(10))
	g.Connect(testV(8), testV(11))
	g.Connect(testV(9), testV(11))
	g.Connect(testV(10), testV(11))

	start := make(Set)
	start.Add(testV(2))
	start.Add(testV(1))
	reverse := make(Set)
	reverse.Add(testV(11))
	reverse.Add(testV(6))

	t.Run("DepthFirst", func(t *testing.T) {
		var visits []vertexAtDepth
		g.walk(depthFirst|downOrder, true, start, func(v Vertex, d int) error {
			visits = append(visits, vertexAtDepth{v, d})
			return nil

		})
		expect := []vertexAtDepth{
			{testV(2), 0}, {testV(5), 1}, {testV(7), 2}, {testV(9), 3}, {testV(11), 4}, {testV(8), 3}, {testV(10), 3}, {testV(4), 1}, {testV(1), 0}, {testV(3), 1}, {testV(6), 2},
		}
		if !reflect.DeepEqual(visits, expect) {
			t.Errorf("expected visits:\n%v\ngot:\n%v\n", expect, visits)
		}
	})
	t.Run("ReverseDepthFirst", func(t *testing.T) {
		var visits []vertexAtDepth
		g.walk(depthFirst|upOrder, true, reverse, func(v Vertex, d int) error {
			visits = append(visits, vertexAtDepth{v, d})
			return nil

		})
		expect := []vertexAtDepth{
			{testV(6), 0}, {testV(3), 1}, {testV(1), 2}, {testV(11), 0}, {testV(9), 1}, {testV(7), 2}, {testV(5), 3}, {testV(2), 4}, {testV(4), 3}, {testV(8), 1}, {testV(10), 1},
		}
		if !reflect.DeepEqual(visits, expect) {
			t.Errorf("expected visits:\n%v\ngot:\n%v\n", expect, visits)
		}
	})
	t.Run("BreadthFirst", func(t *testing.T) {
		var visits []vertexAtDepth
		g.walk(breadthFirst|downOrder, true, start, func(v Vertex, d int) error {
			visits = append(visits, vertexAtDepth{v, d})
			return nil

		})
		expect := []vertexAtDepth{
			{testV(1), 0}, {testV(2), 0}, {testV(3), 1}, {testV(4), 1}, {testV(5), 1}, {testV(6), 2}, {testV(7), 2}, {testV(10), 3}, {testV(8), 3}, {testV(9), 3}, {testV(11), 4},
		}
		if !reflect.DeepEqual(visits, expect) {
			t.Errorf("expected visits:\n%v\ngot:\n%v\n", expect, visits)
		}
	})
	t.Run("ReverseBreadthFirst", func(t *testing.T) {
		var visits []vertexAtDepth
		g.walk(breadthFirst|upOrder, true, reverse, func(v Vertex, d int) error {
			visits = append(visits, vertexAtDepth{v, d})
			return nil

		})
		expect := []vertexAtDepth{
			{testV(11), 0}, {testV(6), 0}, {testV(10), 1}, {testV(8), 1}, {testV(9), 1}, {testV(3), 1}, {testV(7), 2}, {testV(1), 2}, {testV(4), 3}, {testV(5), 3}, {testV(2), 4},
		}
		if !reflect.DeepEqual(visits, expect) {
			t.Errorf("expected visits:\n%v\ngot:\n%v\n", expect, visits)
		}
	})

	t.Run("TopologicalOrder", func(t *testing.T) {
		order := g.topoOrder(downOrder)

		// Validate the order by checking it against the initial graph. We only
		// need to verify that each node has it's direct dependencies
		// satisfied.
		completed := map[Vertex]bool{}
		for _, v := range order {
			deps := g.DownEdges(v)
			for _, dep := range deps {
				if !completed[dep] {
					t.Fatalf("walking node %v, but dependency %v was not yet seen", v, dep)
				}
			}
			completed[v] = true
		}
	})
	t.Run("ReverseTopologicalOrder", func(t *testing.T) {
		order := g.topoOrder(upOrder)

		// Validate the order by checking it against the initial graph. We only
		// need to verify that each node has it's direct dependencies
		// satisfied.
		completed := map[Vertex]bool{}
		for _, v := range order {
			deps := g.UpEdges(v)
			for _, dep := range deps {
				if !completed[dep] {
					t.Fatalf("walking node %v, but dependency %v was not yet seen", v, dep)
				}
			}
			completed[v] = true
		}
	})
}

const testGraphTransReductionStr = `
1
  2
2
  3
3
`

const testGraphTransReductionMoreStr = `
1
  2
2
  3
3
  4
4
`

const testGraphTransReductionMultipleRootsStr = `
1
  2
2
  3
3
  4
4
5
  6
6
  7
7
  8
8
`
