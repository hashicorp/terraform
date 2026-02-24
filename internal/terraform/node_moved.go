// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type movedAnalysisRuntime struct {
	Collector *moveStatementsCollector
}

type movedExecutionRuntime struct {
	CrossTypeMover *refactoring.CrossTypeMover
	Collector      *moveResultsCollector
}

type nodeExpandMoved struct {
	Stmt    *refactoring.MoveStatement
	Index   int
	Runtime *movedAnalysisRuntime
}

var (
	_ GraphNodeDynamicExpandable = (*nodeExpandMoved)(nil)
	_ GraphNodeReferenceable     = (*nodeExpandMoved)(nil)
	_ GraphNodeReferencer        = (*nodeExpandMoved)(nil)
	_ graphNodeEvalContextScope  = (*nodeExpandMoved)(nil)
)

func (n *nodeExpandMoved) Name() string {
	if n == nil || n.Stmt == nil {
		return fmt.Sprintf("moved[%d] (expand)", n.Index)
	}
	return fmt.Sprintf("moved[%d]: %s -> %s (expand)", n.Index, n.Stmt.From, n.Stmt.To)
}

func (n *nodeExpandMoved) ModulePath() addrs.Module {
	if n.Stmt == nil {
		return addrs.RootModule
	}
	if len(n.Stmt.DeclModule) > 0 {
		return n.Stmt.DeclModule
	}
	if n.Stmt.From != nil {
		return n.Stmt.From.Module()
	}
	return addrs.RootModule
}

func (n *nodeExpandMoved) Path() evalContextScope {
	return evalContextModuleInstance{
		Addr: n.ModulePath().UnkeyedInstanceShim(),
	}
}

func (n *nodeExpandMoved) References() []*addrs.Reference {
	if n.Stmt == nil || n.Stmt.ForEach == nil {
		return []*addrs.Reference{}
	}

	refs, _ := langrefs.ReferencesInExpr(addrs.ParseRef, n.Stmt.ForEach)
	return refs
}

func (n *nodeExpandMoved) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{}
}

func (n *nodeExpandMoved) MoveOrderingStatement() *refactoring.MoveStatement {
	if n == nil {
		return nil
	}
	return n.Stmt
}

func (n *nodeExpandMoved) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	expandedStmts, diags := n.expandStatements(ctx)
	if diags.HasErrors() {
		return nil, diags
	}

	if n.Runtime != nil && n.Runtime.Collector != nil {
		for i, stmt := range expandedStmts {
			n.Runtime.Collector.Record(n.Index, i, stmt)
		}
	}
	return nil, diags
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
	if n.Runtime == nil {
		return nil
	}
	return refactoring.ApplySingleMoveStatement(
		n.Stmt,
		ctx.State(),
		n.Runtime.CrossTypeMover,
		n.Runtime.Collector.RecordOldAddr,
		n.Runtime.Collector.RecordBlockage,
	)
}

func (n *nodeExpandMoved) expandStatements(ctx EvalContext) ([]refactoring.MoveStatement, tfdiags.Diagnostics) {
	if n == nil || n.Stmt == nil {
		return nil, nil
	}
	if n.Stmt.ForEach == nil {
		return []refactoring.MoveStatement{*n.Stmt}, nil
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(validateMovedForEachReferences(n.Stmt.ForEach))
	if diags.HasErrors() {
		return nil, diags
	}

	modPath := n.ModulePath()
	if len(modPath) == 0 {
		return n.expandStatementsForModuleInstance(ctx)
	}

	type perModuleExpansion struct {
		module addrs.ModuleInstance
		stmts  []refactoring.MoveStatement
	}
	var expansions []perModuleExpansion

	forEachModuleInstance(ctx.InstanceExpander(), modPath, false, func(module addrs.ModuleInstance) {
		moduleCtx := evalContextForModuleInstance(ctx, module)
		stmts, moreDiags := n.expandStatementsForModuleInstance(moduleCtx)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return
		}
		expansions = append(expansions, perModuleExpansion{
			module: module,
			stmts:  stmts,
		})
	}, func(pem addrs.PartialExpandedModule) {
		diags = diags.Append(movedForEachUnknownModuleInstancesDiag(n.Stmt, pem))
	})
	if diags.HasErrors() {
		return nil, diags
	}

	if len(expansions) == 0 {
		return nil, nil
	}

	// A moved statement ultimately executes as a static MoveStatement and thus
	// must expand to the same concrete move set for every instance of the
	// declaring module.
	baselineKeys := movedStatementSetKeys(expansions[0].stmts)
	for i := 1; i < len(expansions); i++ {
		if !equalStringSlices(baselineKeys, movedStatementSetKeys(expansions[i].stmts)) {
			diags = diags.Append(movedForEachInconsistentAcrossModuleInstancesDiag(n.Stmt, expansions[0].module, expansions[i].module))
			return nil, diags
		}
	}

	return dedupeMoveStatements(expansions[0].stmts), diags
}

func (n *nodeExpandMoved) expandStatementsForModuleInstance(ctx EvalContext) ([]refactoring.MoveStatement, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	forEachMap, known, forEachDiags := evaluateForEachExpression(n.Stmt.ForEach, ctx, false)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() || !known {
		return nil, diags
	}

	keys := make([]string, 0, len(forEachMap))
	for k := range forEachMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := make([]refactoring.MoveStatement, 0, len(keys))
	for _, k := range keys {
		repData := instances.RepetitionData{
			EachKey:   cty.StringVal(k),
			EachValue: forEachMap[k],
		}
		stmt, stmtDiags := expandMoveStatementTemplate(n.Stmt, repData)
		diags = diags.Append(stmtDiags)
		if stmtDiags.HasErrors() {
			continue
		}
		ret = append(ret, stmt)
	}
	if diags.HasErrors() {
		return nil, diags
	}
	return ret, diags
}

func validateMovedForEachReferences(expr hcl.Expression) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if expr == nil {
		return diags
	}
	refs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, expr)
	diags = diags.Append(moreDiags)
	for _, ref := range refs {
		switch ref.Subject.(type) {
		case addrs.InputVariable, addrs.LocalValue:
			// Supported in step 2.
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each reference in moved block",
				Detail:   "The `for_each` expression in a `moved` block may reference only input variables and local values.",
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}
	return diags
}

func expandMoveStatementTemplate(stmt *refactoring.MoveStatement, keyData instances.RepetitionData) (refactoring.MoveStatement, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if stmt == nil {
		return refactoring.MoveStatement{}, diags
	}
	if stmt.FromExpr == nil || stmt.ToExpr == nil {
		return refactoring.MoveStatement{}, movedTemplateInternalErrorDiag(stmt, "moved statement template is missing endpoint expressions")
	}

	fromTraversal, fromDiags := exprToTraversalWithRepetitionData(
		stmt.FromExpr,
		keyData,
		`Only "each.key" and "each.value" can be used in moved address index expressions.`,
		"Moved address index expression cannot be sensitive.",
	)
	diags = diags.Append(fromDiags)
	toTraversal, toDiags := exprToTraversalWithRepetitionData(
		stmt.ToExpr,
		keyData,
		`Only "each.key" and "each.value" can be used in moved address index expressions.`,
		"Moved address index expression cannot be sensitive.",
	)
	diags = diags.Append(toDiags)
	if diags.HasErrors() {
		return refactoring.MoveStatement{}, diags
	}

	fromRel, parseFromDiags := addrs.ParseMoveEndpoint(fromTraversal)
	diags = diags.Append(parseFromDiags)
	toRel, parseToDiags := addrs.ParseMoveEndpoint(toTraversal)
	diags = diags.Append(parseToDiags)
	if diags.HasErrors() {
		return refactoring.MoveStatement{}, diags
	}

	declModule := stmt.DeclModule
	fromAbs, toAbs := addrs.UnifyMoveEndpoints(declModule, fromRel, toRel)
	if fromAbs == nil || toAbs == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid \"moved\" addresses",
			Detail:   "The `from` and `to` addresses must either both refer to resources or both refer to modules.",
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return refactoring.MoveStatement{}, diags
	}

	ret := *stmt
	ret.From = fromAbs
	ret.To = toAbs
	ret.ForEach = nil
	ret.FromExpr = nil
	ret.ToExpr = nil
	return ret, diags
}

func movedTemplateInternalErrorDiag(stmt *refactoring.MoveStatement, detail string) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var subject *hcl.Range
	if stmt != nil {
		subject = stmt.DeclRange.ToHCL().Ptr()
	}

	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid moved block template",
		Detail:   detail,
		Subject:  subject,
	})
	return diags
}

func movedForEachUnknownModuleInstancesDiag(stmt *refactoring.MoveStatement, module addrs.PartialExpandedModule) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var subject *hcl.Range
	if stmt != nil {
		subject = stmt.DeclRange.ToHCL().Ptr()
	}
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Invalid for_each argument",
		Detail:   fmt.Sprintf("Terraform cannot evaluate the `moved` block `for_each` expression for %s because the set of declaring module instances is not yet known.", module),
		Subject:  subject,
	})
	return diags
}

func movedForEachInconsistentAcrossModuleInstancesDiag(stmt *refactoring.MoveStatement, a, b addrs.ModuleInstance) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var subject *hcl.Range
	if stmt != nil {
		subject = stmt.DeclRange.ToHCL().Ptr()
	}
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Inconsistent moved block for_each across module instances",
		Detail:   fmt.Sprintf("The `for_each` expression in this `moved` block expands differently for %s and %s. A `moved` block declared in a module must expand to the same set of moves for all instances of that module.", a, b),
		Subject:  subject,
	})
	return diags
}

func movedStatementSetKeys(stmts []refactoring.MoveStatement) []string {
	ret := make([]string, 0, len(stmts))
	for _, stmt := range stmts {
		ret = append(ret, movedStatementKey(&stmt))
	}
	sort.Strings(ret)
	return ret
}

func dedupeMoveStatements(stmts []refactoring.MoveStatement) []refactoring.MoveStatement {
	if len(stmts) < 2 {
		return stmts
	}
	seen := make(map[string]struct{}, len(stmts))
	ret := make([]refactoring.MoveStatement, 0, len(stmts))
	for _, stmt := range stmts {
		key := movedStatementKey(&stmt)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		ret = append(ret, stmt)
	}
	return ret
}

func movedStatementKey(stmt *refactoring.MoveStatement) string {
	if stmt == nil || stmt.From == nil || stmt.To == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(stmt.From.String())
	b.WriteString(" -> ")
	b.WriteString(stmt.To.String())
	return b.String()
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
