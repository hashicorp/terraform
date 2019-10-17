package projectlang

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// DynamicEvaluateData is an interface used during dynamic evaluation operations
// to interrogate information from the configuration and other value sources.
//
// DynamicEvaluateData is used only in situations where it's assume that
// references in the expressions were pre-validated to ensure that they
// refer to items in the configuration, so DynamicEvaluateData implementations
// can assume that all requested objects should exist in configuration, and
// panic if that does not hold in practice.
type DynamicEvaluateData interface {
	// BaseDir returns the directory that should be considered as the base
	// directory for any relative filesystem paths that appear in expressions.
	BaseDir() string

	// LocalValueExpr returns the expression associated with the given named
	// local value.
	//
	// While the caller is responsible for determining final values for most
	// other referenceable objects exposed from DynamicEvaluateData, local
	// values are treated as part of the language itself and their expressions
	// are evaluated by the language runtime in the projectlang package to
	// ensure that each one is only evaluated once while processing a single
	// expression evaluation call.
	LocalValueExpr(addrs.LocalValue) hcl.Expression

	// ContextValue returns the value associated with the given context key
	// in the current project execution context.
	ContextValue(addrs.ProjectContextValue) cty.Value

	// WorkspaceConfigValue returns a value representing a particular workspace
	// configuration when accessed in expressions.
	//
	// For a workspace configuration block that does not have for_each set,
	// the return value is an object whose attributes are the output values
	// of the workspace in question.
	//
	// For a workspace configuration block that does have for_each set, there
	// result has an extra nesting level where the top-level object attributes
	// are the workspace instance keys and each instance's output values
	// appear in nested objects.
	WorkspaceConfigValue(addrs.ProjectWorkspaceConfig) cty.Value
}

// DynamicEvaluateEach represents the current "each" repetition when evaluating
// expressions.
//
// If evaluating in a context where no "for_each" is active, use
// projectlang.NoEach as a placeholder value.
type DynamicEvaluateEach struct {
	Key   addrs.InstanceKey
	Value cty.Value
}

// ValueObj returns the "each" object value that should represent the reciever
// in HCL expression evaluation.
func (e DynamicEvaluateEach) ValueObj() cty.Value {
	switch k := e.Key.(type) {
	case nil:
		return cty.NullVal(cty.Object(map[string]cty.Type{
			"key":   cty.DynamicPseudoType,
			"value": cty.DynamicPseudoType,
		}))
	case addrs.StringKey:
		return cty.ObjectVal(map[string]cty.Value{
			"key":   cty.StringVal(string(k)),
			"value": e.Value,
		})
	case addrs.IntKey:
		return cty.ObjectVal(map[string]cty.Value{
			"key":   cty.NumberIntVal(int64(k)),
			"value": e.Value,
		})
	default:
		panic(fmt.Sprintf("unsupported key type %T", e.Key))
	}
}

// NoEach is the zero value of DynamicEvaluateEach and is used to represent
// situations where no "each" object is available.
var NoEach = DynamicEvaluateEach{
	Key:   addrs.NoKey,
	Value: cty.NilVal,
}

// DynamicEvaluateExprs is the full expression evaluation pass that supports
// references to any of the object types in the project configuration language.
//
// The given DynamicEvaluateData and DynamicEvaluateEach values together
// provide the data that will be available for use in reference expressions.
// For expressions where the "each" object should not be available, set
// that argument to projectlang.NoEach.
//
// Grouping multiple evaluations together both allows us to avoid re-evaluating
// common local values multiple times and, more importantly, ensures that we'll
// only report errors for each expression once rather than repeating them once
// per expression. Callers should therefore prefer to gather together all of
// their dynamic evaluation expressions into a single call and avoid combining
// diagnostics from separate calls to DynamicEvaluateExprs in the same output.
func DynamicEvaluateExprs(exprs []hcl.Expression, data DynamicEvaluateData, each DynamicEvaluateEach) ([]cty.Value, tfdiags.Diagnostics) {
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
		refs := findReferencesInExpr(expr)
		ctx, moreDiags := dynamicEvalContext(refs, data, each, localValues)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			ret[i] = cty.DynamicVal
			continue
		}
		val, hclDiags := expr.Value(ctx)
		diags = diags.Append(hclDiags)
		ret[i] = val
	}

	return ret, diags
}

func dynamicEvalContext(refs []*addrs.ProjectConfigReference, data DynamicEvaluateData, each DynamicEvaluateEach, localValues map[string]cty.Value) (*hcl.EvalContext, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	currentWorkspaces := map[string]cty.Value{}
	upstreamWorkspaces := map[string]cty.Value{}
	contextValues := map[string]cty.Value{}
	eachVal := each.ValueObj()

	for _, ref := range refs {
		switch addr := ref.Subject.(type) {
		case addrs.ProjectWorkspaceConfig:
			obj := data.WorkspaceConfigValue(addr)
			switch addr.Rel {
			case addrs.ProjectWorkspaceCurrent:
				currentWorkspaces[addr.Name] = obj
			case addrs.ProjectWorkspaceUpstream:
				upstreamWorkspaces[addr.Name] = obj
			}
		case addrs.ProjectContextValue:
			contextValues[addr.Name] = data.ContextValue(addr)
		case addrs.LocalValue:
			// Nothing to do for these because they should already be in
			// localValues.
		case addrs.ForEachAttr:
			// Nothing to do for these because we've already populated
			// eachVal above.
		default:
			panic(fmt.Sprintf("unsupported reference type %T", addr))
		}
	}

	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local":     cty.ObjectVal(localValues),
			"workspace": cty.ObjectVal(currentWorkspaces),
			"upstream":  cty.ObjectVal(upstreamWorkspaces),
			"context":   cty.ObjectVal(contextValues),
			"each":      eachVal,
		},
		Functions: lang.Functions(false, data.BaseDir()),
	}, diags
}
