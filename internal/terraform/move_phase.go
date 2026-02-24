// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// MoveExecutionPlan is the immutable handoff from move analysis to move
// execution. Analysis expands templates, normalizes the concrete move set, and
// freezes execution order before any state mutation occurs.
type MoveExecutionPlan struct {
	ConcreteStatements []refactoring.MoveStatement
	OrderedStatements  []refactoring.MoveStatement
}

func (c *Context) runPrePlanMovePhase(config *configs.Config, prevRunState *states.State, rootVariableValues InputValues) ([]refactoring.MoveStatement, refactoring.MoveResults, tfdiags.Diagnostics) {
	plan, diags := c.analyzeMovePhase(config, prevRunState, rootVariableValues)
	if diags.HasErrors() {
		return plan.ConcreteStatements, refactoring.MakeMoveResults(), diags
	}

	moveResults, execDiags := c.executeMovePhase(config, prevRunState, plan)
	diags = diags.Append(execDiags)
	return plan.ConcreteStatements, moveResults, diags
}

func (c *Context) analyzeMovePhase(config *configs.Config, prevRunState *states.State, rootVariableValues InputValues) (MoveExecutionPlan, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := MoveExecutionPlan{
		ConcreteStatements: nil,
		OrderedStatements:  nil,
	}

	explicitMoveStmts := refactoring.FindMoveStatements(config)
	expandedExplicit := explicitMoveStmts

	if moveStatementsUseForEach(explicitMoveStmts) {
		expandedExplicit, diags = c.expandMoveStatementsViaAnalysisGraph(config, prevRunState, rootVariableValues, explicitMoveStmts)
		if diags.HasErrors() {
			ret.ConcreteStatements = expandedExplicit
			return ret, diags
		}
	}

	// Compatibility rule: derive implied moves only after explicit template
	// expansion so suppression runs against concrete explicit addresses.
	implicitMoveStmts := refactoring.ImpliedMoveStatements(config, prevRunState, expandedExplicit)

	concrete := normalizeMoveStatementsStable(expandedExplicit, implicitMoveStmts)
	ret.ConcreteStatements = concrete
	if len(concrete) == 0 {
		return ret, nil
	}

	ordered, orderDiags := refactoring.OrderedMoveStatements(concrete, nil)
	if orderDiags.HasErrors() {
		// Preserve ApplyMoves behavior: invalid ordering graphs are skipped here
		// and reported later by postPlanValidateMoves.
		log.Printf("[ERROR] analyzeMovePhase: %s", orderDiags.ErrWithWarnings())
		return ret, nil
	}
	ret.OrderedStatements = ordered
	return ret, nil
}

func (c *Context) expandMoveStatementsViaAnalysisGraph(config *configs.Config, prevRunState *states.State, rootVariableValues InputValues, stmts []refactoring.MoveStatement) ([]refactoring.MoveStatement, tfdiags.Diagnostics) {
	if len(stmts) == 0 {
		return nil, nil
	}

	stmtCollector := newMoveStatementsCollector()
	graph, diags := (&MovedAnalysisGraphBuilder{
		Statements:         stmts,
		Config:             config,
		RootVariableValues: rootVariableValues,
		Runtime: &movedAnalysisRuntime{
			Collector: stmtCollector,
		},
	}).Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		return nil, diags
	}

	walker, walkDiags := c.walk(graph, walkEval, &graphWalkOpts{
		InputState: prevRunState,
		Config:     config,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)
	// Analysis-only walk still allocates state wrappers internally.
	walker.State.Close()
	if diags.HasErrors() {
		return nil, diags
	}

	return stmtCollector.Results(), diags
}

func (c *Context) executeMovePhase(config *configs.Config, prevRunState *states.State, plan MoveExecutionPlan) (refactoring.MoveResults, tfdiags.Diagnostics) {
	if len(plan.OrderedStatements) == 0 {
		return refactoring.MakeMoveResults(), nil
	}

	collector := newMoveResultsCollector()
	crossTypeMover := refactoring.NewCrossTypeMover(c.plugins.ProviderFactories())

	graph, diags := (&MovedExecutionGraphBuilder{
		OrderedStatements: plan.OrderedStatements,
		Runtime: &movedExecutionRuntime{
			CrossTypeMover: crossTypeMover,
			Collector:      collector,
		},
	}).Build(addrs.RootModuleInstance)
	if diags.HasErrors() {
		diags = diags.Append(crossTypeMover.Close())
		return refactoring.MakeMoveResults(), diags
	}

	walker, walkDiags := c.walk(graph, walkEval, &graphWalkOpts{
		InputState: prevRunState,
		Config:     config,
	})
	diags = diags.Append(walker.NonFatalDiagnostics)
	diags = diags.Append(walkDiags)

	// c.walk uses a deep-copied state wrapper, so copy the resulting state data
	// back into our caller-owned previous-run state.
	newState := walker.State.Close()
	*prevRunState = *newState

	diags = diags.Append(crossTypeMover.Close())
	return collector.Results(), diags
}

func normalizeMoveStatementsStable(explicit, implied []refactoring.MoveStatement) []refactoring.MoveStatement {
	total := len(explicit) + len(implied)
	if total == 0 {
		return nil
	}

	stmts := make([]refactoring.MoveStatement, 0, total)
	stmts = append(stmts, explicit...)
	stmts = append(stmts, implied...)
	return dedupeMoveStatements(stmts)
}

func moveStatementsUseForEach(stmts []refactoring.MoveStatement) bool {
	for _, stmt := range stmts {
		if stmt.ForEach != nil {
			return true
		}
	}
	return false
}
