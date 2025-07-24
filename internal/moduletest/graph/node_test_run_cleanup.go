// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package graph

import (
	"fmt"
	"log"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest"
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
	references, _ := n.run.GetReferences()

	for _, run := range n.priorRuns {
		// we'll also draw an implicit reference to all prior runs to make sure
		// they execute first
		references = append(references, &addrs.Reference{
			Subject:     run.Addr(),
			SourceRange: tfdiags.SourceRangeFromHCL(n.run.Config.DeclRange),
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

	state := ctx.GetFileState(n.run.Config.StateKey)
	if state == nil {
		// then we don't have a state for this run block in the manifest, which
		// is okay - it means the states were partially cleaned up last time.
		//
		// we set nothing for this, on this basis that this since this state was
		// successfully cleaned up so any state that might have relied on this
		// one would have also been cleaned up so it should not be needed.
		return
	}

	outputs := make(map[string]cty.Value)
	for name, output := range state.State.RootOutputValues {
		if output.Sensitive {
			outputs[name] = output.Value.Mark(marks.Sensitive)
			continue
		}
		outputs[name] = output.Value
	}
	n.run.Outputs = cty.ObjectVal(outputs)

	ctx.AddRunBlock(n.run)
}
