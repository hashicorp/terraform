// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/dag"
)

// testGraphnotContains is an assertion helper that tests that a node is
// NOT contained in the graph.
func testGraphNotContains(t *testing.T, g *Graph, name string) {
	for _, v := range g.Vertices() {
		if dag.VertexName(v) == name {
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
	for _, v := range g.Vertices() {
		if dag.VertexName(v) == B {
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
	for _, v := range g.Ancestors(vertexB) {
		if dag.VertexName(v) == A {
			// Success
			return
		}
	}

	t.Fatalf(
		"Expected %q before %q in:\n\n%s",
		A, B, g.String())
}
