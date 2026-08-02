// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

// Edge represents an edge in the graph, with a source and target vertex.
type Edge struct {
	S, T Vertex
}
