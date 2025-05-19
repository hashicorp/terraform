// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GraphBuilder is an interface that can be implemented and used with
// Terraform to build the graph that Terraform walks.
type GraphBuilder interface {
	// Build builds the graph for the given module path. It is up to
	// the interface implementation whether this build should expand
	// the graph or not.
	Build(addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics)
}

// BasicGraphBuilder is a GraphBuilder that builds a graph out of a
// series of transforms and (optionally) validates the graph is a valid
// structure.
type BasicGraphBuilder struct {
	Steps []GraphTransformer
	// Optional name to add to the graph debug log
	Name string

	// SkipGraphValidation indicates whether the graph validation (enabled by default)
	// should be skipped after the graph is built.
	SkipGraphValidation bool
}

func (b *BasicGraphBuilder) Build(path addrs.ModuleInstance) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	g := &Graph{Path: path}

	var lastStepStr string
	for _, step := range b.Steps {
		if step == nil {
			continue
		}
		log.Printf("[TRACE] Executing graph transform %T", step)

		err := step.Transform(g)
		if thisStepStr := g.StringWithNodeTypes(); thisStepStr != lastStepStr {
			log.Printf("[TRACE] Completed graph transform %T with new graph:\n%s  ------", step, logging.Indent(thisStepStr))
			lastStepStr = thisStepStr
		} else {
			log.Printf("[TRACE] Completed graph transform %T (no changes)", step)
		}

		if err != nil {
			if nf, isNF := err.(tfdiags.NonFatalError); isNF {
				diags = diags.Append(nf.Diagnostics)
			} else if diag, isDiag := err.(tfdiags.DiagnosticsAsError); isDiag {
				diags = diags.Append(diag.Diagnostics)
				return g, diags
			} else {
				diags = diags.Append(err)
				return g, diags
			}
		}
	}

	// Return early if the graph validation is skipped
	// This behavior is currently only used by the graph command
	// which only wants to display the dot representation of the graph
	if b.SkipGraphValidation {
		return g, diags
	}

	if err := g.Validate(); err != nil {
		log.Printf("[ERROR] Graph validation failed. Graph:\n\n%s", g.String())
		diags = diags.Append(err)
		return nil, diags
	}

	return g, diags
}
