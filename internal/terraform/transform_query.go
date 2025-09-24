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

	// if validate is true, we are in validate mode and should not exclude any resources.
	validate bool
}

func (t *QueryTransformer) Transform(g *Graph) error {
	var nodesToRemove []dag.Vertex
	for v := range dag.SelectSeq[GraphNodeConfigResource](g.VerticesSeq()) {
		mode := v.ResourceAddr().Resource.Mode
		var shouldRemove bool
		switch {
		// if we are validating lists, we validate all resources
		case t.validate && t.includeLists:
			shouldRemove = false

		// we are in default validate mode, but do not want to include list resources
		case t.validate && !t.includeLists:
			shouldRemove = false

		// We are planning list resources, so we should remove all non-list resources and their dependencies.
		case mode != addrs.ListResourceMode && t.includeLists:
			shouldRemove = true

		// We are planning non-list resources, so we should remove all list resources and their dependencies.
		case mode == addrs.ListResourceMode && !t.includeLists:
			shouldRemove = true
		}

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
