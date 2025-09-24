// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

type QueryTransformer struct {
	// includeLists is a flag that determines whether list resources should be included in the query.
	// If true, list resources and their dependencies will be included in the query. If false, list resources and their dependencies will be excluded.
	includeLists bool
}

func (t *QueryTransformer) Transform(g *Graph) error {
	var nodesToRemove []dag.Vertex
	for v := range dag.SelectSeq[GraphNodeConfigResource](g.VerticesSeq()) {
		mode := v.ResourceAddr().Resource.Mode
		// The first condition checks if we want to include list resources, in which case we should remove all
		// non-list resources.
		// The second condition checks if we want to exclude list resources, in which case we should remove all
		// list resources.
		shouldRemove := (mode != addrs.ListResourceMode && t.includeLists) ||
			(mode == addrs.ListResourceMode && !t.includeLists)

		// If the node is to be removed, we need to remove it and its descendants from the graph.
		if shouldRemove {
			deps := g.Descendants(v)
			deps.Add(v)
			for node := range deps {
				nodesToRemove = append(nodesToRemove, node)
			}
		}
	}

	for _, node := range nodesToRemove {
		g.Remove(node)
	}

	return nil
}
