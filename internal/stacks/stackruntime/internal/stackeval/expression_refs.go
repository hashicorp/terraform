package stackeval

import (
	"context"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Referer is implemented by types that have expressions that can refer to
// [Referenceable] objects.
type Referer interface {
	// References returns descriptions of all of the expression references
	// made from the configuration of the receiver.
	References(ctx context.Context) []stackaddrs.AbsReference
}

// ReferencesInExpr returns all of the valid references contained in the given
// HCL expression.
//
// It ignores any invalid references, on the assumption that the expression
// will eventually be evaluated and then those invalid references would be
// reported as errors at that point.
func ReferencesInExpr(ctx context.Context, expr hcl.Expression) []stackaddrs.Reference {
	if expr == nil {
		return nil
	}
	return referencesInTraverals(ctx, expr.Variables())
}

// ReferencesInBody returns all of the valid references contained in the given
// HCL body.
//
// It ignores any invalid references, on the assumption that the body
// will eventually be evaluated and then those invalid references would be
// reported as errors at that point.
func ReferencesInBody(ctx context.Context, body hcl.Body, spec hcldec.Spec) []stackaddrs.Reference {
	if body == nil {
		return nil
	}
	return referencesInTraverals(ctx, hcldec.Variables(body, spec))
}

func referencesInTraverals(ctx context.Context, traversals []hcl.Traversal) []stackaddrs.Reference {
	if len(traversals) == 0 {
		return nil
	}
	ret := make([]stackaddrs.Reference, 0, len(traversals))
	for _, traversal := range traversals {
		ref, _, moreDiags := stackaddrs.ParseReference(traversal)
		if moreDiags.HasErrors() {
			// We'll ignore any traversals that are not valid references,
			// on the assumption that we'd catch them during a subsequent
			// evaluation of the same expression/body/etc.
			continue
		}
		ret = append(ret, ref)
	}
	return ret
}

func makeReferencesAbsolute(localRefs []stackaddrs.Reference, stackAddr stackaddrs.StackInstance) []stackaddrs.AbsReference {
	if len(localRefs) == 0 {
		return nil
	}
	ret := make([]stackaddrs.AbsReference, 0, len(localRefs))
	for _, localRef := range localRefs {
		// contextual refs require a more specific scope than an entire
		// stack, so they can't be represented as [AbsReference].
		if _, isContextual := localRef.Target.(stackaddrs.ContextualRef); isContextual {
			continue
		}
		ret = append(ret, localRef.Absolute(stackAddr))
	}
	return ret
}

// requiredComponentsForReferer is the main underlying implementation
// of Applyable.RequiredComponents, allowing the types which directly implement
// that interface to worry only about their own unique way of gathering up
// the relevant references from their configuration, since the work of
// peeling away references until we've found all of the components is the
// same regardless of where the references came from.
//
// This is a best-effort which will produce a complete result only if the
// configuration is completely valid. If not, the result is likely to be
// incomplete, which we accept on the assumption that the invalidity would
// also make the resulting plan non-applyable and thus it doesn't actually
// matter what the required components are.
func (m *Main) requiredComponentsForReferer(ctx context.Context, obj Referer, phase EvalPhase) collections.Set[stackaddrs.AbsComponent] {
	ret := collections.NewSet[stackaddrs.AbsComponent]()
	initialRefs := obj.References(ctx)

	// queued tracks objects we've previously queued -- which may or may not
	// still be in the queue -- so that we can avoid re-visiting the same
	// object multiple times and thus ensure the following loop will definitely
	// eventually terminate, even in the presence of reference cycles, because
	// the number of unique reference addresses in the configuration is
	// finite.
	queued := collections.NewSet[stackaddrs.AbsReferenceable]()
	queue := make([]stackaddrs.AbsReferenceable, len(initialRefs))
	for i, ref := range initialRefs {
		queue[i] = ref.Target()
		queued.Add(queue[i])
	}

	for len(queue) != 0 {
		targetAddr, remain := queue[0], queue[1:]
		queue = remain

		// If this is a direct reference to a component then we can just
		// add it and continue.
		if componentAddr, ok := targetAddr.Item.(stackaddrs.Component); ok {
			ret.Add(stackaddrs.AbsComponent{
				Stack: targetAddr.Stack,
				Item:  componentAddr,
			})
			continue
		}

		// For all other address types, we need to find the corresponding
		// object and, if it's also Applyable, ask it for its references.
		//
		// For all of the fallible situations below, we'll just skip over
		// this item on failure, because it's not this function's responsibility
		// to report problems with the configuration.
		//
		// Since we're going to ignore all errors anyway, we can safely use
		// a reference with no source location information.
		ref := stackaddrs.AbsReference{
			Stack: targetAddr.Stack,
			Ref: stackaddrs.Reference{
				Target: targetAddr.Item,
			},
		}
		target, _ := m.ResolveAbsExpressionReference(ctx, ref, phase)
		if target == nil {
			continue
		}
		targetReferer, ok := target.(Referer)
		if !ok {
			// Anything that isn't a referer cannot possibly indirectly
			// refer to a component.
			continue
		}
		for _, newRef := range targetReferer.References(ctx) {
			newTargetAddr := newRef.Target()
			if !queued.Has(newTargetAddr) {
				queue = append(queue, newTargetAddr)
				queued.Add(newTargetAddr)
			}
		}
	}

	return ret
}
