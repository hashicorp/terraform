// Copyright (c) HashiCorp, Inc.
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

func TestAcyclicGraphRoot(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(3, 1))

	if root, err := g.Root(); err != nil {
		t.Fatalf("err: %s", err)
	} else if root != 3 {
		t.Fatalf("bad: %#v", root)
	}
}

func TestAcyclicGraphRoot_cycle(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(3, 1))

	if _, err := g.Root(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphRoot_multiple(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))

	if _, err := g.Root(); err == nil {
		t.Fatal("should error")
	}
}

func TestAyclicGraphTransReduction(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(2, 3))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestAyclicGraphTransReduction_more(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(1, 4))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(2, 4))
	g.Connect(BasicEdge(3, 4))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionMoreStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestAyclicGraphTransReduction_multipleRoots(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(1, 4))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(2, 4))
	g.Connect(BasicEdge(3, 4))

	g.Add(5)
	g.Add(6)
	g.Add(7)
	g.Add(8)
	g.Connect(BasicEdge(5, 6))
	g.Connect(BasicEdge(5, 7))
	g.Connect(BasicEdge(5, 8))
	g.Connect(BasicEdge(6, 7))
	g.Connect(BasicEdge(6, 8))
	g.Connect(BasicEdge(7, 8))
	g.TransitiveReduction()

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphTransReductionMultipleRootsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

// use this to simulate slow sort operations
type counter struct {
	Name  string
	Calls int64
}

func (s *counter) String() string {
	s.Calls++
	return s.Name
}

// Make sure we can reduce a sizable, fully-connected graph.
func TestAyclicGraphTransReduction_fullyConnected(t *testing.T) {
	var g AcyclicGraph

	const nodeCount = 200
	nodes := make([]*counter, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = &counter{Name: strconv.Itoa(i)}
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
			g.Connect(BasicEdge(nodes[i], nodes[j]))
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
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(3, 1))

	if err := g.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestAcyclicGraphValidate_cycle(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(3, 1))
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 1))

	if err := g.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphValidate_cycleSelf(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Connect(BasicEdge(1, 1))

	if err := g.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestAcyclicGraphAncestors(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Connect(BasicEdge(0, 1))
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(3, 4))
	g.Connect(BasicEdge(4, 5))

	actual := g.Ancestors(2)

	expected := []Vertex{3, 4, 5}

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
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Connect(BasicEdge(0, 1))
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 3))
	g.Connect(BasicEdge(3, 4))
	g.Connect(BasicEdge(4, 5))

	actual := g.Descendants(2)

	expected := []Vertex{0, 1}

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
	g.Add(0)
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Add(6)
	g.Connect(BasicEdge(0, 1))
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(2, 6))
	g.Connect(BasicEdge(3, 4))
	g.Connect(BasicEdge(4, 5))
	g.Connect(BasicEdge(5, 6))

	actual := g.FirstDescendantsWith(6, func(v Vertex) bool {
		// looking for first odd descendants
		return v.(int)%2 != 0
	})

	expected := make(Set)
	expected.Add(1)
	expected.Add(5)

	if expected.Intersection(actual).Len() != expected.Len() {
		t.Fatalf("expected %#v, got %#v\n", expected, actual)
	}

	foundOne := g.MatchDescendant(6, func(v Vertex) bool {
		return v.(int) == 1
	})
	if !foundOne {
		t.Fatal("did not match 1 in the graph")
	}

	foundSix := g.MatchDescendant(6, func(v Vertex) bool {
		return v.(int) == 6
	})
	if foundSix {
		t.Fatal("6 should not be a descendant of itself")
	}

	foundTen := g.MatchDescendant(6, func(v Vertex) bool {
		return v.(int) == 10
	})
	if foundTen {
		t.Fatal("10 is not in the graph at all")
	}
}

func TestAcyclicGraphFindAncestors(t *testing.T) {
	var g AcyclicGraph
	g.Add(0)
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Add(5)
	g.Add(6)
	g.Connect(BasicEdge(1, 0))
	g.Connect(BasicEdge(2, 1))
	g.Connect(BasicEdge(6, 2))
	g.Connect(BasicEdge(4, 3))
	g.Connect(BasicEdge(5, 4))
	g.Connect(BasicEdge(6, 5))

	actual := g.FirstAncestorsWith(6, func(v Vertex) bool {
		// looking for first odd ancestors
		return v.(int)%2 != 0
	})

	expected := make(Set)
	expected.Add(1)
	expected.Add(5)

	if expected.Intersection(actual).Len() != expected.Len() {
		t.Fatalf("expected %#v, got %#v\n", expected, actual)
	}

	foundOne := g.MatchAncestor(6, func(v Vertex) bool {
		return v.(int) == 1
	})
	if !foundOne {
		t.Fatal("did not match 1 in the graph")
	}

	foundSix := g.MatchAncestor(6, func(v Vertex) bool {
		return v.(int) == 6
	})
	if foundSix {
		t.Fatal("6 should not be a descendant of itself")
	}

	foundTen := g.MatchAncestor(6, func(v Vertex) bool {
		return v.(int) == 10
	})
	if foundTen {
		t.Fatal("10 is not in the graph at all")
	}
}

func TestAcyclicGraphWalk(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(3, 1))

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
		{1, 2, 3},
		{2, 1, 3},
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
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)
	g.Connect(BasicEdge(4, 3))
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(2, 1))

	var visits []Vertex
	var lock sync.Mutex
	err := g.Walk(func(v Vertex) tfdiags.Diagnostics {
		lock.Lock()
		defer lock.Unlock()

		var diags tfdiags.Diagnostics

		if v == 2 {
			diags = diags.Append(fmt.Errorf("error"))
			return diags
		}

		visits = append(visits, v)
		return diags
	})
	if err == nil {
		t.Fatal("should error")
	}

	expected := []Vertex{1}
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
		for i := 0; i < count; i++ {
			g.Add(fmt.Sprintf("A%d", i))
		}

		// layer B
		for i := 0; i < count; i++ {
			B := fmt.Sprintf("B%d", i)
			g.Add(B)
			for j := 0; j < count; j++ {
				g.Connect(BasicEdge(B, fmt.Sprintf("A%d", j)))
			}
		}

		// layer C
		for i := 0; i < count; i++ {
			c := fmt.Sprintf("C%d", i)
			g.Add(c)
			for j := 0; j < count; j++ {
				// connect them to previous layers so we have something that requires reduction
				g.Connect(BasicEdge(c, fmt.Sprintf("A%d", j)))
				g.Connect(BasicEdge(c, fmt.Sprintf("B%d", j)))
			}
		}

		// layer D
		for i := 0; i < count; i++ {
			d := fmt.Sprintf("D%d", i)
			g.Add(d)
			for j := 0; j < count; j++ {
				g.Connect(BasicEdge(d, fmt.Sprintf("A%d", j)))
				g.Connect(BasicEdge(d, fmt.Sprintf("B%d", j)))
				g.Connect(BasicEdge(d, fmt.Sprintf("C%d", j)))
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
		g.Add(i)
	}
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(1, 4))
	g.Connect(BasicEdge(2, 4))
	g.Connect(BasicEdge(2, 5))
	g.Connect(BasicEdge(3, 6))
	g.Connect(BasicEdge(4, 7))
	g.Connect(BasicEdge(5, 7))
	g.Connect(BasicEdge(7, 8))
	g.Connect(BasicEdge(7, 9))
	g.Connect(BasicEdge(7, 10))
	g.Connect(BasicEdge(8, 11))
	g.Connect(BasicEdge(9, 11))
	g.Connect(BasicEdge(10, 11))

	start := make(Set)
	start.Add(2)
	start.Add(1)
	reverse := make(Set)
	reverse.Add(11)
	reverse.Add(6)

	t.Run("DepthFirst", func(t *testing.T) {
		var visits []vertexAtDepth
		g.walk(depthFirst|downOrder, true, start, func(v Vertex, d int) error {
			visits = append(visits, vertexAtDepth{v, d})
			return nil

		})
		expect := []vertexAtDepth{
			{2, 0}, {5, 1}, {7, 2}, {9, 3}, {11, 4}, {8, 3}, {10, 3}, {4, 1}, {1, 0}, {3, 1}, {6, 2},
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
			{6, 0}, {3, 1}, {1, 2}, {11, 0}, {9, 1}, {7, 2}, {5, 3}, {2, 4}, {4, 3}, {8, 1}, {10, 1},
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
			{1, 0}, {2, 0}, {3, 1}, {4, 1}, {5, 1}, {6, 2}, {7, 2}, {10, 3}, {8, 3}, {9, 3}, {11, 4},
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
			{11, 0}, {6, 0}, {10, 1}, {8, 1}, {9, 1}, {3, 1}, {7, 2}, {1, 2}, {4, 3}, {5, 3}, {2, 4},
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
