// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

// testGraphnotContains is an assertion helper that tests that a node is
// NOT contained in the graph.
func testGraphNotContains(t *testing.T, g *Graph, name string) {
	for v := range g.VerticesSeq() {
		if v.Name() == name {
			t.Fatalf(
				"Expected %q to NOT be in:\n\n%s",
				name, g.String())
		}
	}
}

// testGraphHappensBefore is an assertion helper that tests that node
// A (dag.VertexName value) happens before node B.
func testGraphHappensBefore(t *testing.T, g *Graph, A, B string) {
	t.Helper()
	// Find the B vertex
	var vertexB dag.Vertex
	for v := range g.VerticesSeq() {
		if v.Name() == B {
			vertexB = v
			break
		}
	}
	if vertexB == nil {
		t.Fatalf(
			"Expected %q before %q. Couldn't find %q in:\n\n%s",
			A, B, B, g.String())
	}

	// Look at ancestors
	// Make sure B is in there
	for v := range g.Ancestors(vertexB).All() {
		if v.Name() == A {
			// Success
			return
		}
	}

	t.Fatalf(
		"Expected %q before %q in:\n\n%s",
		A, B, g.String())
}

// Regression for #39441: ResourceGraph must keep the edges of a cycle. The
// resource graph is only used by `terraform graph`, where -draw-cycles skips
// validation and hands us an invalid (cyclic) graph; TransitiveReduction
// assumes a DAG and can drop the edges that form the cycle.
func TestResourceGraphPreservesCycleEdges(t *testing.T) {
	mk := func(name string) *stubConfigResource {
		return &stubConfigResource{addr: addrs.ConfigResource{
			Module: addrs.RootModule,
			Resource: addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "terraform_data",
				Name: name,
			},
		}}
	}
	a, b, c := mk("a"), mk("b"), mk("c")

	g := &Graph{}
	g.Add(a)
	g.Add(b)
	g.Add(c)
	g.Connect(a, c) // A depends on C
	g.Connect(b, a) // B depends on A
	g.Connect(c, b) // C depends on B

	rg := g.ResourceGraph()
	for _, tc := range []struct {
		from, to *stubConfigResource
	}{
		{a, c},
		{b, a},
		{c, b},
	} {
		if !rg.DirectDependenciesOf(tc.from.addr).Has(tc.to.addr) {
			t.Fatalf("ResourceGraph dropped edge %s -> %s", tc.from.addr, tc.to.addr)
		}
	}
}
