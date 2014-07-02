package digraph

import (
	"reflect"
	"sort"
	"testing"
)

func TestStronglyConnectedComponents(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
b -> c
c -> b
c -> d
d -> e`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	sccs := StronglyConnectedComponents(nlist, false)
	if len(sccs) != 4 {
		t.Fatalf("bad: %v", sccs)
	}

	sccs = StronglyConnectedComponents(nlist, true)
	if len(sccs) != 1 {
		t.Fatalf("bad: %v", sccs)
	}

	cycle := sccs[0]
	if len(cycle) != 2 {
		t.Fatalf("bad: %v", sccs)
	}

	cycleNodes := make([]string, len(cycle))
	for i, c := range cycle {
		cycleNodes[i] = c.(*BasicNode).Name
	}
	sort.Strings(cycleNodes)

	expected := []string{"b", "c"}
	if !reflect.DeepEqual(cycleNodes, expected) {
		t.Fatalf("bad: %#v", cycleNodes)
	}
}

func TestStronglyConnectedComponents2(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
b -> d
b -> e
c -> f
c -> g
g -> a
`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	sccs := StronglyConnectedComponents(nlist, true)
	if len(sccs) != 1 {
		t.Fatalf("bad: %v", sccs)
	}

	cycle := sccs[0]
	if len(cycle) != 3 {
		t.Fatalf("bad: %v", sccs)
	}

	cycleNodes := make([]string, len(cycle))
	for i, c := range cycle {
		cycleNodes[i] = c.(*BasicNode).Name
	}
	sort.Strings(cycleNodes)

	expected := []string{"a", "c", "g"}
	if !reflect.DeepEqual(cycleNodes, expected) {
		t.Fatalf("bad: %#v", cycleNodes)
	}
}
