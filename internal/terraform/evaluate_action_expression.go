package terraform

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// evaluateActionExpression expands the hcl.Expression from a resource's list
// action_trigger.actions. Note that if the action uses count or for_each, it's
// using the resource instances count/for_each.
func evaluateActionExpression(expr hcl.Expression, repData instances.RepetitionData) (*addrs.Reference, tfdiags.Diagnostics) {
	var ref *addrs.Reference
	var diags tfdiags.Diagnostics

	traversal, diags := actionExprToTraversal(expr, repData)
	if diags.HasErrors() {
		return nil, diags
	}

	// We now have a static traversal, so we can just turn it into an addrs.Reference.
	ref, ds := addrs.ParseRef(traversal)
	diags = diags.Append(ds)

	return ref, diags
}

// actionExprToTraversal takes an hcl expression limited to the syntax allowed
// in a resource's lifecycle.action_triggers.actions list, and converts it to a
// static traversal. The RepetitionData contains the data necessary to evaluate
// the only allowed variables in the expression, count.index and each.key.
func actionExprToTraversal(expr hcl.Expression, repData instances.RepetitionData) (hcl.Traversal, tfdiags.Diagnostics) {
	var trav hcl.Traversal
	var diags tfdiags.Diagnostics

	switch e := expr.(type) {
	case *hclsyntax.RelativeTraversalExpr:
		t, d := actionExprToTraversal(e.Source, repData)
		diags = diags.Append(d)
		trav = append(trav, t...)
		trav = append(trav, e.Traversal...)

	case *hclsyntax.ScopeTraversalExpr:
		// a static reference, we can just append the traversal
		trav = append(trav, e.Traversal...)

	case *hclsyntax.IndexExpr:
		// Get the collection from the index expression
		t, d := actionExprToTraversal(e.Collection, repData)
		diags = diags.Append(d)
		if diags.HasErrors() {
			return nil, diags
		}
		trav = append(trav, t...)

		// The index key is the only place where we could have variables that
		// reference count and each, so we need to parse those independently.
		idx, hclDiags := parseReplaceTriggeredByKeyExpr(e.Key, repData)
		diags = diags.Append(hclDiags)

		trav = append(trav, idx)

	default:
		// Something unexpected got through config validation. We're not sure
		// what it is, but we'll point it out in the diagnostics for the user
		// to fix.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid action expression",
			Detail:   "Unexpected expression found in action_triggers.actions.",
			Subject:  e.Range().Ptr(),
		})
	}

	return trav, diags
}
