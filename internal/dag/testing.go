// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package dag

// Minimal vertex implementation for testing purposes
//
// NOTE: the receiver for this type is a value, not a pointer.
// This means the string will be used as the unique identifier of the vertex (see `marshalVertexID`).
// If the receiver was a pointer then the memory address would be used as the identifier,
// which would not be stable for testing purposes. Mermaid formatting tests are sensitive to this.
type stringV string

var (
	_ Vertex          = stringV("")
	_ GraphNodeDotter = stringV("")
)

func (v stringV) Name() string {
	return string(v)
}

func (v stringV) DotNode(n string, opts *DotOpts) *DotNode {
	return &DotNode{Name: n, Attrs: map[string]string{}}
}
