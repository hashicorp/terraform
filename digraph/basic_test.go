package digraph

import (
	"fmt"
	"testing"
)

func TestParseBasic(t *testing.T) {
	spec := `a -> b ; first
b -> c ; second
b -> d ; third
z -> a`
	nodes := ParseBasic(spec)
	if len(nodes) != 5 {
		t.Fatalf("bad: %v", nodes)
	}

	a := nodes["a"]
	if a.Name != "a" {
		t.Fatalf("bad: %v", a)
	}
	aEdges := a.Edges()
	if len(aEdges) != 1 {
		t.Fatalf("bad: %v", a.Edges())
	}
	if fmt.Sprintf("%v", aEdges[0]) != "first" {
		t.Fatalf("bad: %v", aEdges[0])
	}

	b := nodes["b"]
	if len(b.Edges()) != 2 {
		t.Fatalf("bad: %v", b.Edges())
	}

	c := nodes["c"]
	if len(c.Edges()) != 0 {
		t.Fatalf("bad: %v", c.Edges())
	}

	d := nodes["d"]
	if len(d.Edges()) != 0 {
		t.Fatalf("bad: %v", d.Edges())
	}

	z := nodes["z"]
	zEdges := z.Edges()
	if len(zEdges) != 1 {
		t.Fatalf("bad: %v", z.Edges())
	}
	if fmt.Sprintf("%v", zEdges[0]) != "Edge" {
		t.Fatalf("bad: %v", zEdges[0])
	}
}
