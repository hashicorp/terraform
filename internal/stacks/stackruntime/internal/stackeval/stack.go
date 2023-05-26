package stackeval

import (
	"context"

	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Stack represents an instance of a [StackConfig] after it's had its
// repetition arguments (if any) evaluated to determine how many instances
// it has.
type Stack struct {
	addr stackaddrs.StackInstance

	main *Main
}

var _ ExpressionScope = (*Stack)(nil)

func (s *Stack) Addr() stackaddrs.StackInstance {
	return s.addr
}

func (s *Stack) IsRoot() bool {
	return s.addr.IsRoot()
}

// ResolveExpressionReference implements ExpressionScope, providing the
// global scope for evaluation within an already-instanciated stack during the
// plan and apply phases.
func (s *Stack) ResolveExpressionReference(ctx context.Context, ref stackaddrs.Reference) (Referenceable, tfdiags.Diagnostics) {
	panic("unimplemented")
}
