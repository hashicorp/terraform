// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraformtest

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/moduletest"
	hcltest "github.com/hashicorp/terraform/internal/moduletest/hcl"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestGraphBuilder is a terraform.GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	File       *moduletest.File
	Config     *configs.Config
	GlobalVars map[string]backendrun.UnparsedVariableValue
}

// See terraform.GraphBuilder
func (b *TestGraphBuilder) Build(path addrs.ModuleInstance) (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "TestGraphBuilder",
	}).Build(path)
}

// See terraform.GraphBuilder
func (b *TestGraphBuilder) Steps() []terraform.GraphTransformer {
	steps := []terraform.GraphTransformer{
		&TestFileTransformer{File: b.File, globalVars: b.GlobalVars, config: b.Config},
		&TestRunTransformer{File: b.File, config: b.Config, globalVars: b.GlobalVars},
		&ConfigTransformer{File: b.File, config: b.Config, globalVars: b.GlobalVars},
		&AttachVariablesTransformer{},
		&ApplyNoParallelTransformer{},
		&CloseTestRootModuleTransformer{},
		&terraform.ReferenceTransformer{},
		&terraform.TransitiveReductionTransformer{},
	}

	return steps
}

func WalkGraph(g *terraform.Graph, cb dag.WalkFunc) tfdiags.Diagnostics {
	return g.AcyclicGraph.Walk(cb)
}

// GraphNodeExecutable is an interface that can be implemented by nodes in the
// graph that can be executed.
type GraphNodeExecutable interface {
	Execute(*hcltest.VariableContext, *terraform.Graph) tfdiags.Diagnostics
}
