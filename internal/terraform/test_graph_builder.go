// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestGraphBuilder is a GraphBuilder implementation that builds a graph for
// a terraform test file. The file may contain multiple runs, and each run may have
// dependencies on other runs.
type TestGraphBuilder struct {
	File *moduletest.File
}

// See GraphBuilder
func (b *TestGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	log.Printf("[TRACE] building graph for terraform test")
	return (&BasicGraphBuilder{
		Steps: b.Steps(),
		Name:  "TestGraphBuilder",
	}).Build(path)
}

// See GraphBuilder
func (b *TestGraphBuilder) Steps() []GraphTransformer {
	steps := []GraphTransformer{
		&TestRunTransformer{File: b.File},
		&ApplyNoParallelTransformer{},
		&CloseTestRootModuleTransformer{},
		&TransitiveReductionTransformer{},
	}

	return steps
}
