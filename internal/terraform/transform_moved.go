// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

type MovedTransformer struct {
	Config  *configs.Config
	Targets []addrs.Targetable

	skip bool
}

func (t *MovedTransformer) Transform(g *Graph) error {
	if t.skip {
		return nil
	}

	node := &NodeExecuteMoved{
		Config:  t.Config,
		Targets: t.Targets,
	}
	g.Add(node)

	for _, v := range g.Vertices() {
		switch v.(type) {
		case graphNodeExpandsInstances:
			// We should move the resources before anything within the graph
			// expands and starts doing real work.
			g.Connect(dag.BasicEdge(v, node))
		case NodeApplyableProvider:
			// We should only try and move the resources after the providers
			// have been initialised.
			g.Connect(dag.BasicEdge(node, v))
		}
	}

	return nil
}
