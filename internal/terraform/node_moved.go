// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type movedExecutionRuntime struct {
	CrossTypeMover *refactoring.CrossTypeMover
	Collector      *moveResultsCollector
}

type nodeExpandMoved struct {
	Stmt    *refactoring.MoveStatement
	Index   int
	Runtime *movedExecutionRuntime
}

var (
	_ GraphNodeDynamicExpandable = (*nodeExpandMoved)(nil)
	_ GraphNodeReferenceable     = (*nodeExpandMoved)(nil)
	_ GraphNodeReferencer        = (*nodeExpandMoved)(nil)
)

func (n *nodeExpandMoved) Name() string {
	return fmt.Sprintf("moved[%d]: %s -> %s (expand)", n.Index, n.Stmt.From, n.Stmt.To)
}

func (n *nodeExpandMoved) ModulePath() addrs.Module {
	return addrs.RootModule
}

func (n *nodeExpandMoved) References() []*addrs.Reference {
	return []*addrs.Reference{}
}

func (n *nodeExpandMoved) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{}
}

func (n *nodeExpandMoved) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	g.Add(&nodeMovedInstance{
		Stmt:    n.Stmt,
		Index:   n.Index,
		Runtime: n.Runtime,
	})
	addRootNodeToGraph(&g)
	return &g, nil
}

type nodeMovedInstance struct {
	Stmt    *refactoring.MoveStatement
	Index   int
	Runtime *movedExecutionRuntime
}

var _ GraphNodeExecutable = (*nodeMovedInstance)(nil)

func (n *nodeMovedInstance) Name() string {
	return fmt.Sprintf("moved[%d]: %s -> %s", n.Index, n.Stmt.From, n.Stmt.To)
}

func (n *nodeMovedInstance) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	_ = op
	return refactoring.ApplySingleMoveStatement(
		n.Stmt,
		ctx.State(),
		n.Runtime.CrossTypeMover,
		n.Runtime.Collector.RecordOldAddr,
		n.Runtime.Collector.RecordBlockage,
	)
}
