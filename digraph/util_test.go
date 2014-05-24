package digraph

import (
	"reflect"
	"testing"
)

func TestDepthFirstWalk(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
d -> f
e -> a ; cycle`)
	root := nodes["a"]
	expected := []string{
		"a",
		"b",
		"e",
		"c",
		"d",
		"f",
	}
	index := 0
	DepthFirstWalk(root, func(n Node) bool {
		name := n.(*BasicNode).Name
		if expected[index] != name {
			t.Fatalf("expected: %v, got %v", expected[index], name)
		}
		index++
		return true
	})
}

func TestInDegree(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	expected := map[string]int{
		"a": 0,
		"b": 1,
		"c": 1,
		"d": 1,
		"e": 2,
		"f": 1,
	}
	indegree := InDegree(nlist)
	for n, d := range indegree {
		name := n.(*BasicNode).Name
		exp := expected[name]
		if exp != d {
			t.Fatalf("Expected %d for %s, got %d",
				exp, name, d)
		}
	}
}

func TestOutDegree(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	expected := map[string]int{
		"a": 3,
		"b": 1,
		"c": 1,
		"d": 1,
		"e": 0,
		"f": 0,
	}
	outDegree := OutDegree(nlist)
	for n, d := range outDegree {
		name := n.(*BasicNode).Name
		exp := expected[name]
		if exp != d {
			t.Fatalf("Expected %d for %s, got %d",
				exp, name, d)
		}
	}
}

func TestSinks(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	sinks := Sinks(nlist)

	var haveE, haveF bool
	for _, n := range sinks {
		name := n.(*BasicNode).Name
		switch name {
		case "e":
			haveE = true
		case "f":
			haveF = true
		}
	}
	if !haveE || !haveF {
		t.Fatalf("missing sink")
	}
}

func TestSources(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f
x -> y`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	sources := Sources(nlist)
	if len(sources) != 2 {
		t.Fatalf("bad: %v", sources)
	}

	var haveA, haveX bool
	for _, n := range sources {
		name := n.(*BasicNode).Name
		switch name {
		case "a":
			haveA = true
		case "x":
			haveX = true
		}
	}
	if !haveA || !haveX {
		t.Fatalf("missing source %v %v", haveA, haveX)
	}
}

func TestUnreachable(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f
f -> a
x -> y
y -> z`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	unreached := Unreachable(nodes["a"], nlist)
	if len(unreached) != 3 {
		t.Fatalf("bad: %v", unreached)
	}

	var haveX, haveY, haveZ bool
	for _, n := range unreached {
		name := n.(*BasicNode).Name
		switch name {
		case "x":
			haveX = true
		case "y":
			haveY = true
		case "z":
			haveZ = true
		}
	}
	if !haveX || !haveY || !haveZ {
		t.Fatalf("missing %v %v %v", haveX, haveY, haveZ)
	}
}

func TestUnreachable2(t *testing.T) {
	nodes := ParseBasic(`a -> b
a -> c
a -> d
b -> e
c -> e
d -> f
f -> a
x -> y
y -> z`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	unreached := Unreachable(nodes["x"], nlist)
	if len(unreached) != 6 {
		t.Fatalf("bad: %v", unreached)
	}

	expected := map[string]struct{}{
		"a": struct{}{},
		"b": struct{}{},
		"c": struct{}{},
		"d": struct{}{},
		"e": struct{}{},
		"f": struct{}{},
	}
	out := map[string]struct{}{}
	for _, n := range unreached {
		name := n.(*BasicNode).Name
		out[name] = struct{}{}
	}

	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("bad: %v %v", out, expected)
	}
}
