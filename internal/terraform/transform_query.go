// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

type QueryTransformer struct {
	// queryPlan is true when we are planning list resources.
	queryPlan bool

	// if validate is true, we are in validate mode and should not exclude any resources.
	validate bool
}

func (t *QueryTransformer) Transform(g *Graph) error {
	if t.validate && t.queryPlan {
		// if we are validating query files, we validate all resources
		return nil
	}

	for v := range dag.SelectSeq[GraphNodeConfigResource](g.VerticesSeq()) {
		mode := v.ResourceAddr().Resource.Mode
		var shouldRemove bool
		switch {
		// we are in default validate mode, we do not want to include list resources
		case t.validate:
			shouldRemove = mode == addrs.ListResourceMode

		// We are planning list resources, so we should remove all non-list resources and their dependencies.
		case t.queryPlan && mode != addrs.ListResourceMode:
			shouldRemove = true

		// We are planning/applying non-list resources, so we should remove all list resources and their dependencies.
		case !t.queryPlan && mode == addrs.ListResourceMode:
			shouldRemove = true
		}

		// If the node is to be removed, we need to remove it and its descendants from the graph.
		if shouldRemove {
			deps := g.Descendants(v)
			g.Remove(v)
			for node := range deps {
				g.Remove(node)
			}
		}
	}
	return nil
}
