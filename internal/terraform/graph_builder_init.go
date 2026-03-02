// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type InitGraphBuilder struct {
	// A config derived from the root module
	Config *configs.Config

	RootVariableValues InputValues

	Walker configs.ModuleWalker
}

// See GraphBuilder
func (b *InitGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform dependencies")
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "InitGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *InitGraphBuilder) Steps() []GraphTransformer {
	steps := []GraphTransformer{}

	if b.Config.Parent == nil {
		steps = append(steps, &RootVariableTransformer{
			Config:    b.Config,
			RawValues: b.RootVariableValues,
		})
	} else {
		steps = append(steps, &ModuleVariableTransformer{
			Config:     b.Config,
			ModuleOnly: true,
		})
	}

	steps = append(steps, []GraphTransformer{
		&ModuleTransformer{
			Config: b.Config,
			Walker: b.Walker,
		},

		&LocalTransformer{
			Config: b.Config,
		},

		&ReferenceTransformer{},

		// Filters out any vertices that aren't relevant to the init graph
		&TransformFilter{
			Keep: func(v dag.Vertex) bool {
				switch n := v.(type) {
				case *nodeInstallModule:
					return true
				case *NodeRootVariable:
					return n.Config.Const
				case *nodeExpandModuleVariable:
					return n.Config.Const
				default:
					return false
				}
			},
		},

		&RootTransformer{},

		&TransitiveReductionTransformer{},
	}...)

	return steps
}
