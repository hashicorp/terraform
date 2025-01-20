// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestGraphBuilder is a GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	File               *moduletest.File
	GlobalVars         map[string]backendrun.UnparsedVariableValue
	ConfigsProviderMap map[string]map[string]bool
}

// See GraphBuilder
func (b *TestGraphBuilder) Build(path addrs.ModuleInstance) (*terraform.Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	return (&terraform.BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "TestGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps() []terraform.GraphTransformer {
	steps := []terraform.GraphTransformer{
		&TestRunTransformer{File: b.File, globalVars: b.GlobalVars},
		&TestConfigTransformer{},
		&TestProvidersTransformer{configsProviderMap: b.ConfigsProviderMap},
		&CloseTestGraphTransformer{},
		&terraform.TransitiveReductionTransformer{},
	}

	return steps
}
