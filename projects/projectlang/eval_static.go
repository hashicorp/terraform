package projectlang

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
)

// StaticEvaluateData is an interface used during static evaluation operations
// to interrogate information from the configuration.
//
// StaticEvaluateData is used only in situations where it's assume that
// references in the expressions were pre-validated to ensure that they
// refer to items in the configuration, so StaticEvaluateData implementations
// can assume that all requested objects should exist, and panic if that does
// not hold in practice.
type StaticEvaluateData interface {
	// BaseDir returns the directory that should be considered as the base
	// directory for any relative filesystem paths that appear in expressions.
	BaseDir() string

	// LocalValueExpr returns the expression associated with the given named
	// local value.
	LocalValueExpr(addrs.LocalValue) hcl.Expression

	// WorkspaceConfigForEach returns the for_each expression associated
	// with the given workspace, or nil if the workspace config does not have
	// the for_each argument set.
	WorkspaceConfigForEachExpr(addrs.ProjectWorkspaceConfig) hcl.Expression
}

// StaticEvaluateExprs is a limited evaluation pass that can recursively resolve
// local values but will return errors if the expression directly or indirectly
// refers to any workspace outputs or context values.
//
// This should be used for parts of the project configuration that need to
// be known _before_ context values are set or existing workspace states can be
// read.
//
// The given StaticEvaluateData will be used to obtain the expression for any
// local value that the given expression refers to. This function assumes
// that references in the expression have already been validated to ensure
// that they refer to objects that actually exist in configuration, so this
// function may panic if that contract is not upheld.
//
// Grouping multiple evaluations together both allows us to avoid re-evaluating
// common local values multiple times and, more importantly, ensures that we'll
// only report errors for each expression once rather than repeating them once
// per expression. Callers should therefore prefer to gather together all of
// their static evaluation expressions into a single call and avoid combining
// diagnostics from separate calls to StaticEvaluateExprs in the same output.
func StaticEvaluateExprs(exprs []hcl.Expression, data StaticEvaluateData) ([]cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	dependents := make(map[addrs.LocalValue][]addrs.LocalValue)
	inDegree := make(map[addrs.LocalValue]int)
	var queue []addrs.LocalValue

	// We'll seed our queue with the references in the given expressions themselves.
	for _, expr := range exprs {
		for _, traversal := range expr.Variables() {
			ref, moreDiags := addrs.ParseProjectConfigRef(traversal)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}
			addr, ok := ref.Subject.(addrs.LocalValue)
			if ok {
				queue = append(queue, addr)
				dependents[addr] = nil
				inDegree[addr] = 0
			}
		}
	}

	// Now we'll work our way through the graph of locals to find all of
	// the ones we directly and indirectly depend on.
	localValues := make(map[string]cty.Value)
	for i := 0; i < len(queue); i++ { // queue length will grow during iteration
		addr := queue[i]
		if _, exists := localValues[addr.Name]; exists {
			// We already dealt with this one via another path through the graph.
			continue
		}
		// We'll make a placeholder element for now, just so we know we've visited
		// this one, and then overwrite it with a real value later.
		localValues[addr.Name] = cty.NilVal

		expr := data.LocalValueExpr(addr)
		if expr == nil {
			// Should never happen because references should be validated by our caller.
			panic(fmt.Sprintf("no expression available for %s", addr))
		}
		for _, traversal := range expr.Variables() {
			ref, moreDiags := addrs.ParseProjectConfigRef(traversal)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}

			refAddr, ok := ref.Subject.(addrs.LocalValue)
			if !ok {
				// If it refers to anything other than local values then
				// it's a dynamic local value and so we can't use it
				// in static evaluation.
				localValues[addr.Name] = cty.DynamicVal
				continue
			}
			inDegree[refAddr]++
			dependents[addr] = append(dependents[addr], refAddr)
			queue = append(queue, refAddr)
		}
	}

	// We now need to re-visit all of the addresses in topological order,
	// evaluating the locals as we go. We'll re-use the backing buffer of
	// our queue above, since we know it has sufficient capacity for all
	// of the local values involved.
	queue = queue[:0]
	for addr := range dependents { // Seed queue with locals that have no dependencies
		if inDegree[addr] == 0 {
			queue = append(queue, addr)
		}
	}
	for len(queue) > 0 {
		var addr addrs.LocalValue
		addr, queue = queue[0], queue[1:] // dequeue next item
		if val := localValues[addr.Name]; val != cty.NilVal {
			continue // Already dealt with this one
		}
		delete(inDegree, addr)
		expr := data.LocalValueExpr(addr)
		if expr == nil {
			// Should never happen because references should be validated by our caller.
			panic(fmt.Sprintf("no expression available for %s", addr))
		}
		val, moreDiags := expr.Value(staticEvalContext(data.BaseDir(), localValues))
		diags = diags.Append(moreDiags)
		localValues[addr.Name] = val

		for _, referrerAddr := range dependents[addr] {
			inDegree[referrerAddr]--
			if inDegree[referrerAddr] < 1 {
				queue = append(queue, referrerAddr)
			}
		}
	}

	if len(inDegree) > 0 {
		// TODO: This error needs to be _much_ better
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Dependency cycle in project configuration",
			Detail:   "There is at least one dependency cycle between the local values in the project configuration.",
		})
		return nil, diags
	}

	// Finally, with all of the local values evaluated, we can evaluate the
	// expressions we were given.
	ret := make([]cty.Value, len(exprs))
	for i, expr := range exprs {
		val, moreDiags := expr.Value(staticEvalContext(data.BaseDir(), localValues))
		diags = diags.Append(moreDiags)
		ret[i] = val

		if !val.IsKnown() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid expression",
				Detail:   fmt.Sprintf("This argument requires a result that can be determined without direct or indirect reference to any context values or workspace outputs."),
				Subject:  expr.Range().Ptr(),
			})
		}
	}

	return ret, diags
}

func staticEvalContext(baseDir string, localValues map[string]cty.Value) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": cty.MapVal(localValues),

			// All of the other top-level objects are just placeholders here
			// so we can still do partial type checking of derived expressions.
			"workspace": cty.DynamicVal,
			"upstream":  cty.DynamicVal,
			"context":   cty.DynamicVal,
		},
		Functions: lang.Functions(true, "."),
	}
}
