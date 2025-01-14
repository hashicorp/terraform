// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/terraform"
)

// ConfigVariablesToOthersTransformer is a GraphTransformer that ensures that each config variable node
// is connected to other relevant nodes in the graph.
type ConfigVariablesToOthersTransformer struct {
}

func (t *ConfigVariablesToOthersTransformer) Transform(g *terraform.Graph) error {
	for _, v := range g.Vertices() {
		node, ok := v.(*nodeConfigVariable)
		if !ok {
			continue
		}

		for _, other := range g.Vertices() {
			if _, ok := other.(*nodeConfigVariable); ok {
				continue
			}

			switch other := other.(type) {
			case *nodeFileVariable:
				g.Connect(dag.BasicEdge(node, other))
			case *nodeGlobalVariable:
				// Only connect the global variable if it is referenced in the config
				_, ok := node.config.Module.Variables[other.Addr.Name]
				if ok {
					g.Connect(dag.BasicEdge(node, other))
				}
			case *nodeRunVariable:
				// Only connect the config variable if it belongs to the same module as the run
				if node.config.Module.SourceDir == other.config.Module.SourceDir && node.run == other.run {
					g.Connect(dag.BasicEdge(node, other))
				}
			}
		}
	}
	return nil
}
