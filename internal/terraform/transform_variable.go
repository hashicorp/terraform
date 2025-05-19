// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

// RootVariableTransformer is a GraphTransformer that adds all the root
// variables to the graph.
//
// Root variables are currently no-ops but they must be added to the
// graph since downstream things that depend on them must be able to
// reach them.
type RootVariableTransformer struct {
	Config *configs.Config

	RawValues InputValues

	// Planning must be set to true when building a planning graph, and must be
	// false when building an apply graph.
	Planning bool

	// DestroyApply must be set to true when applying a destroy operation and
	// false otherwise.
	DestroyApply bool
}

func (t *RootVariableTransformer) Transform(g *Graph) error {
	// We can have no variables if we have no config.
	if t.Config == nil {
		return nil
	}

	// We're only considering root module variables here, since child
	// module variables are handled by ModuleVariableTransformer.
	vars := t.Config.Module.Variables

	// Add all variables here
	for _, v := range vars {
		node := &NodeRootVariable{
			Addr: addrs.InputVariable{
				Name: v.Name,
			},
			Config:       v,
			RawValue:     t.RawValues[v.Name],
			Planning:     t.Planning,
			DestroyApply: t.DestroyApply,
		}
		g.Add(node)
	}

	return nil
}
