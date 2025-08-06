// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest"
	teststates "github.com/hashicorp/terraform/internal/moduletest/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeExecutable    = (*NodeTestRunCleanup)(nil)
	_ GraphNodeReferenceable = (*NodeTestRunCleanup)(nil)
	_ GraphNodeReferencer    = (*NodeTestRunCleanup)(nil)
)

type NodeTestRunCleanup struct {
	run       *moduletest.Run
	priorRuns map[string]*moduletest.Run
	opts      *graphOptions
}

func (n *NodeTestRunCleanup) Name() string {
	return fmt.Sprintf("%s.%s (cleanup)", n.opts.File.Name, n.run.Addr().String())
}

func (n *NodeTestRunCleanup) References() []*addrs.Reference {
	references, _ := moduletest.GetRunReferences(n.run.Config)

	for _, run := range n.priorRuns {
		// we'll also draw an implicit reference to all prior runs to make sure
		// they execute first
		references = append(references, &addrs.Reference{
			Subject:     run.Addr(),
			SourceRange: tfdiags.SourceRangeFromHCL(n.run.Config.DeclRange),
		})
	}

	for name, variable := range n.run.ModuleConfig.Module.Variables {

		// because we also draw implicit references back to any variables
		// defined in the test file with the same name as actual variables, then
		// we'll count these as references as well.

		if _, ok := n.run.Config.Variables[name]; ok {

			// BUT, if the variable is defined within the list of variables
			// within the run block then we don't want to draw an implicit
			// reference as the data comes from that expression.

			continue
		}

		references = append(references, &addrs.Reference{
			Subject:     addrs.InputVariable{Name: name},
			SourceRange: tfdiags.SourceRangeFromHCL(variable.DeclRange),
		})
	}

	return references
}

func (n *NodeTestRunCleanup) Referenceable() addrs.Referenceable {
	return n.run.Addr()
}

func (n *NodeTestRunCleanup) Execute(ctx *EvalContext) {
	log.Printf("[TRACE] TestFileRunner: executing run block %s/%s", n.opts.File.Name, n.run.Name)

	n.run.Status = moduletest.Pass

	state, err := ctx.LoadState(n.run.Config)
	if err != nil {
		n.run.Status = moduletest.Fail
		n.run.Diagnostics = n.run.Diagnostics.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to load state",
			Detail:   fmt.Sprintf("Could not retrieve state for run %s: %s.", n.run.Name, err),
			Subject:  n.run.Config.Backend.DeclRange.Ptr(),
		})
		return
	}

	outputs := make(map[string]cty.Value)
	for name, output := range state.RootOutputValues {
		if output.Sensitive {
			outputs[name] = output.Value.Mark(marks.Sensitive)
			continue
		}
		outputs[name] = output.Value
	}
	n.run.Outputs = cty.ObjectVal(outputs)

	ctx.SetFileState(n.run.Config.StateKey, n.run, state, teststates.StateReasonNone)
	ctx.AddRunBlock(n.run)
}
