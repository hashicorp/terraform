// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
)

type GraphNodeReferenceable interface {
	Referenceable() addrs.Referenceable
}

type GraphNodeReferencer interface {
	References() []*addrs.Reference
}

var _ terraform.GraphTransformer = (*ReferenceTransformer)(nil)

type ReferenceTransformer struct{}

func (r *ReferenceTransformer) Transform(graph *terraform.Graph) error {
	nodes := addrs.MakeMap[addrs.Referenceable, dag.Vertex]()
	for referenceable := range graph.VerticesSeq() {
		if referenceable, ok := referenceable.(GraphNodeReferenceable); ok {
			nodes.Put(referenceable.Referenceable(), referenceable)
			continue
		}
	}

	for node := range graph.VerticesSeq() {
		if referencer, ok := node.(GraphNodeReferencer); ok {
			for _, reference := range referencer.References() {
				if target, ok := nodes.GetOk(reference.Subject); ok {
					graph.Connect(dag.BasicEdge(referencer, target))
				}
			}
		}
	}

	return nil
}
