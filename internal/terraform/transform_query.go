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
	if !t.queryPlan {
		// if we are not running a query-specific operation, we don't need to transform the graph
		// as query-related files would not have been part of the parsed config.
		return nil
	}

	if t.validate {
		// if we are validating query files, we validate all resources
		return nil
	}

	for v := range dag.SelectSeq[GraphNodeConfigResource](g.VerticesSeq()) {
		// we only get here if we are building a query plan, but not validating.
		// Because the config would contain resource blocks from traditional .tf files,
		// we need to exclude them from the plan graph.
		// If the node is to be removed, we need to remove it and its descendants from the graph.
		if v.ResourceAddr().Resource.Mode != addrs.ListResourceMode {
			deps := g.Descendants(v)
			g.Remove(v)
			for node := range deps {
				g.Remove(node)
			}
		}
	}
	return nil
}
