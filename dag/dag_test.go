package dag

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

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

	actual, err := g.Ancestors(2)
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

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

func TestAcyclicGraphDescendents(t *testing.T) {
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

	actual, err := g.Descendents(2)
	if err != nil {
		t.Fatalf("err: %#v", err)
	}

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

func TestAcyclicGraphWalk(t *testing.T) {
	var g AcyclicGraph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(3, 2))
	g.Connect(BasicEdge(3, 1))

	var visits []Vertex
	var lock sync.Mutex
	err := g.Walk(func(v Vertex) error {
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
	err := g.Walk(func(v Vertex) error {
		lock.Lock()
		defer lock.Unlock()

		if v == 2 {
			return fmt.Errorf("error")
		}

		visits = append(visits, v)
		return nil
	})
	if err == nil {
		t.Fatal("should error")
	}

	expected := [][]Vertex{
		{1},
	}
	for _, e := range expected {
		if reflect.DeepEqual(visits, e) {
			return
		}
	}

	t.Fatalf("bad: %#v", visits)
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
