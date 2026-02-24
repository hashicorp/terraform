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
	if !moveStatementUsesRepetition(n.Stmt) {
		return []refactoring.MoveStatement{*n.Stmt}, nil
	}

	var diags tfdiags.Diagnostics
	diags = diags.Append(validateMovedRepetitionReferences(n.Stmt))
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
		diags = diags.Append(movedRepetitionUnknownModuleInstancesDiag(n.Stmt, pem))
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
			diags = diags.Append(movedRepetitionInconsistentAcrossModuleInstancesDiag(n.Stmt, expansions[0].module, expansions[i].module))
			return nil, diags
		}
	}

	return dedupeMoveStatements(expansions[0].stmts), diags
}

func (n *nodeExpandMoved) expandStatementsForModuleInstance(ctx EvalContext) ([]refactoring.MoveStatement, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	repDataList, repDiags := n.repetitionDataForModuleInstance(ctx)
	diags = diags.Append(repDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	ret := make([]refactoring.MoveStatement, 0, len(repDataList))
	for _, repData := range repDataList {
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

func (n *nodeExpandMoved) repetitionDataForModuleInstance(ctx EvalContext) ([]instances.RepetitionData, tfdiags.Diagnostics) {
	if n == nil || n.Stmt == nil {
		return nil, nil
	}

	if n.Stmt.ForEach != nil {
		return movedForEachRepetitionData(n.Stmt.ForEach, ctx)
	}
	if n.Stmt.Count != nil {
		return movedCountRepetitionData(n.Stmt.Count, ctx)
	}
	return nil, nil
}

func movedForEachRepetitionData(expr hcl.Expression, ctx EvalContext) ([]instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	forEachMap, known, forEachDiags := evaluateForEachExpression(expr, ctx, false)
	diags = diags.Append(forEachDiags)
	if diags.HasErrors() || !known {
		return nil, diags
	}

	keys := make([]string, 0, len(forEachMap))
	for k := range forEachMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ret := make([]instances.RepetitionData, 0, len(keys))
	for _, k := range keys {
		ret = append(ret, instances.RepetitionData{
			EachKey:   cty.StringVal(k),
			EachValue: forEachMap[k],
		})
	}
	return ret, diags
}

func movedCountRepetitionData(expr hcl.Expression, ctx EvalContext) ([]instances.RepetitionData, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	count, countDiags := evaluateCountExpression(expr, ctx, false)
	diags = diags.Append(countDiags)
	if diags.HasErrors() {
		return nil, diags
	}
	if count < 0 {
		// Unknown counts should already produce diagnostics above because
		// allowUnknown=false, but guard here to avoid silent expansion.
		return nil, diags
	}

	ret := make([]instances.RepetitionData, 0, count)
	for i := 0; i < count; i++ {
		ret = append(ret, instances.RepetitionData{
			CountIndex: cty.NumberIntVal(int64(i)),
		})
	}
	return ret, diags
}

func moveStatementUsesRepetition(stmt *refactoring.MoveStatement) bool {
	if stmt == nil {
		return false
	}
	return stmt.ForEach != nil || stmt.Count != nil
}

func movedRepetitionKindLabel(stmt *refactoring.MoveStatement) string {
	switch {
	case stmt != nil && stmt.ForEach != nil:
		return "for_each"
	case stmt != nil && stmt.Count != nil:
		return "count"
	default:
		return "repetition"
	}
}

func validateMovedRepetitionReferences(stmt *refactoring.MoveStatement) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if stmt == nil {
		return diags
	}

	var expr hcl.Expression
	switch {
	case stmt.ForEach != nil:
		expr = stmt.ForEach
	case stmt.Count != nil:
		expr = stmt.Count
	}
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
				Summary:  fmt.Sprintf("Invalid %s reference in moved block", movedRepetitionKindLabel(stmt)),
				Detail:   fmt.Sprintf("The `%s` expression in a `moved` block may reference only input variables and local values.", movedRepetitionKindLabel(stmt)),
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
		`Only "count.index", "each.key", and "each.value" can be used in moved address index expressions.`,
		"Moved address index expression cannot be sensitive.",
	)
	diags = diags.Append(fromDiags)
	toTraversal, toDiags := exprToTraversalWithRepetitionData(
		stmt.ToExpr,
		keyData,
		`Only "count.index", "each.key", and "each.value" can be used in moved address index expressions.`,
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
	ret.Count = nil
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

func movedRepetitionUnknownModuleInstancesDiag(stmt *refactoring.MoveStatement, module addrs.PartialExpandedModule) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var subject *hcl.Range
	if stmt != nil {
		subject = stmt.DeclRange.ToHCL().Ptr()
	}
	kind := movedRepetitionKindLabel(stmt)
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  fmt.Sprintf("Invalid %s argument", kind),
		Detail:   fmt.Sprintf("Terraform cannot evaluate the `moved` block `%s` expression for %s because the set of declaring module instances is not yet known.", kind, module),
		Subject:  subject,
	})
	return diags
}

func movedRepetitionInconsistentAcrossModuleInstancesDiag(stmt *refactoring.MoveStatement, a, b addrs.ModuleInstance) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var subject *hcl.Range
	if stmt != nil {
		subject = stmt.DeclRange.ToHCL().Ptr()
	}
	kind := movedRepetitionKindLabel(stmt)
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  fmt.Sprintf("Inconsistent moved block %s across module instances", kind),
		Detail:   fmt.Sprintf("The `%s` expression in this `moved` block expands differently for %s and %s. A `moved` block declared in a module must expand to the same set of moves for all instances of that module.", kind, a, b),
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
