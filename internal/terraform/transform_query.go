// Copyright IBM Corp. 2014, 2026
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
	if !t.queryPlan {
		// if we are not running a query-specific operation, we don't need to transform the graph
		// as query-related files would not have been part of the parsed config.
		return nil
	}

	if t.validate {
		// if we are validating query files, we validate all resources
		return nil
	}

	// a set to hold the resources that we want to keep and vertices along its path.
	keep := dag.Set{}

	for v := range dag.SelectSeq[GraphNodeConfigResource](g.VerticesSeq()) {
		// we only get here if we are building a query plan, but not validating.
		//
		// By now, the graph already contains all resources from the config, including non-list resources.
		// We start from each list resource node, look at its ancestors, and keep all vertices along its path.
		if v.ResourceAddr().Resource.Mode == addrs.ListResourceMode {
			keep.Add(v)
			deps := g.Ancestors(v)
			for node := range deps {
				keep.Add(node)
			}
		}
	}

	// Remove all nodes that are not in the keep set.
	for v := range g.VerticesSeq() {
		if _, ok := keep[v]; !ok {
			g.Remove(v)
		}
	}

	return nil
}
