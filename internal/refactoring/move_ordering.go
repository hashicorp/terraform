// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import "github.com/hashicorp/terraform/internal/tfdiags"

// MoveOrderingPolicy determines whether one move statement depends on another
// for execution ordering purposes.
type MoveOrderingPolicy interface {
	DependsOn(depender, dependee *MoveStatement) bool
}

// MoveOrderingPolicyFunc adapts a function to [MoveOrderingPolicy].
type MoveOrderingPolicyFunc func(depender, dependee *MoveStatement) bool

func (f MoveOrderingPolicyFunc) DependsOn(depender, dependee *MoveStatement) bool {
	return f(depender, dependee)
}

// DefaultMoveOrderingPolicy applies Terraform's current move chaining/nesting
// ordering semantics.
type DefaultMoveOrderingPolicy struct{}

func (DefaultMoveOrderingPolicy) DependsOn(depender, dependee *MoveStatement) bool {
	return StatementDependsOn(depender, dependee)
}

func moveOrderingPolicyOrDefault(policy MoveOrderingPolicy) MoveOrderingPolicy {
	if policy == nil {
		return DefaultMoveOrderingPolicy{}
	}
	return policy
}

// OrderedMoveStatements returns the statements in dependency-safe execution
// order, using the provided policy or the default policy when nil.
func OrderedMoveStatements(stmts []MoveStatement, policy MoveOrderingPolicy) ([]MoveStatement, tfdiags.Diagnostics) {
	if len(stmts) == 0 {
		return nil, nil
	}

	g := buildMoveStatementGraphWithPolicy(stmts, policy)
	if diags := validateMoveStatementGraph(g); diags.HasErrors() {
		return nil, diags
	}

	g.TransitiveReduction()
	order := g.ReverseTopologicalOrder()
	ret := make([]MoveStatement, 0, len(order))
	for _, raw := range order {
		ret = append(ret, *raw.(*MoveStatement))
	}
	return ret, nil
}
