// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"

	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Referrer is implemented by types that have expressions that can refer to
// [Referenceable] objects.
type Referrer interface {
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
	return referencesInTraversals(ctx, expr.Variables())
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
	return referencesInTraversals(ctx, hcldec.Variables(body, spec))
}

func referencesInTraversals(ctx context.Context, traversals []hcl.Traversal) []stackaddrs.Reference {
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

// requiredComponentsForReferrer is the main underlying implementation
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
func (m *Main) requiredComponentsForReferrer(ctx context.Context, obj Referrer, phase EvalPhase) collections.Set[stackaddrs.AbsComponent] {
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

		// A stack call reference is also special, as we now want all the
		// components of this stack call to be added to the queue as well.
		// This doesn't happen automatically with the references as stack calls
		// do not have a direct reference to their internal components (it
		// actually goes the other way).
		if stackCallAddr, ok := targetAddr.Item.(stackaddrs.StackCall); ok {
			// We're just adding all the components within the stack to the
			// queue. We could be a bit clever if, for example, the reference
			// is to an output of the stack call. We could only add the
			// components needed by that output. This is an okay compromise for
			// now, in which the apply will wait for the whole stack to finish
			// before moving on.
			currentStack := m.Stack(ctx, targetAddr.Stack, phase)
			for step, nextStack := range currentStack.childStacks {
				if step.Name != stackCallAddr.Name {
					// Then this child stack isn't from the current stack call.
					continue
				}

				for _, component := range nextStack.Components(ctx) {
					ref := stackaddrs.AbsReferenceable{
						Stack: component.addr.Stack,
						Item: stackaddrs.Component{
							Name: component.addr.Item.Name,
						},
					}
					if !queued.Has(ref) {
						queue = append(queue, ref)
						queued.Add(ref)
					}
				}

				// We'll also include any other stack calls within the embedded
				// stack.
				for _, call := range nextStack.EmbeddedStackCalls(ctx) {
					ref := stackaddrs.AbsReferenceable{
						Stack: call.addr.Stack,
						Item:  call.addr.Item,
					}
					if !queued.Has(ref) {
						queue = append(queue, ref)
						queued.Add(ref)
					}
				}
			}

			// We don't continue here, as we still want to add anything that
			// the stack call references below.
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
		targetReferrer, ok := target.(Referrer)
		if !ok {
			// Anything that isn't a referer cannot possibly indirectly
			// refer to a component.
			continue
		}
		for _, newRef := range targetReferrer.References(ctx) {
			newTargetAddr := newRef.Target()
			if !queued.Has(newTargetAddr) {
				queue = append(queue, newTargetAddr)
				queued.Add(newTargetAddr)
			}
		}
	}

	return ret
}

// ValidateDependsOn is a helper function that can be used to validate the
// DependsOn field of a component or an embedded stack. It returns diagnostics
// for any invalid references.
//
// The StackConfig argument should be the stack that the component or embedded
// stack is a part of. It is used to validate any references actually exist.
func ValidateDependsOn(ctx context.Context, source *StackConfig, traversals []hcl.Traversal) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, traversal := range traversals {
		// We don't actually care about the result here, only that it has no
		// errors.
		ref, rest, moreDiags := stackaddrs.ParseReference(traversal)
		if moreDiags.HasErrors() {
			diags = diags.Append(moreDiags)
			continue
		}

		switch addr := ref.Target.(type) {
		case stackaddrs.StackCall:
			// Make sure this stack call exists.
			if source.StackCall(ctx, addr) == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid depends_on target",
					Detail:   fmt.Sprintf("The depends_on reference %q does not exist.", addr),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
			}
		case stackaddrs.Component:
			// Make sure this component exists.
			if source.Component(ctx, addr) == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid depends_on target",
					Detail:   fmt.Sprintf("The depends_on reference %q does not exist.", addr),
					Subject:  ref.SourceRange.ToHCL().Ptr(),
				})
			}
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid depends_on target",
				Detail:   fmt.Sprintf("The depends_on argument must refer to an embedded stack or component, but this reference refers to %q.", addr),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
			continue // don't do the rest of the checks
		}

		if len(rest) > 0 {
			// for now, we can only reference components and stacks in
			// configuration, and not instances of them or outputs from them.
			// eg. component.self is valid, but component.self[0] is not.
			//
			// we'll add a warning, as we don't want users thinking the
			// dependency is more precise than it is. But, we'll allow the
			// reference as we can still use it just by ignoring the rest.
			//
			// FIXME: Allowing more fine grained references requires updating
			//   the requiredComponentsForReferrer function (above) to support
			//   AbsComponentInstance instead of AbsComponent. This is a
			//   potentially large refactor, and so only worth it for good
			//   reason and this isn't really that.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Non-valid depends_on target",
				Detail:   fmt.Sprintf(DependsOnDeepReferenceDetail, ref.Target),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return diags
}

var (
	DependsOnDeepReferenceDetail = strings.TrimSpace(`
The depends_on argument should refer directly to an embedded stack or component in configuration, but this reference is too deep.

Terraform Stacks has simplified the reference to the nearest valid target, %q. To remove this warning, update the configuration to the same target.
`)
)
