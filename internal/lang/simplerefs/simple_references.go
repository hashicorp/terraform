// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package simplerefs

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/lang/globalref"
)

// ReferenceGraph is a thread-safe map of global references that supports
// resolution via a custom selector function.
//
// The selector function is called for each reference to determine if it should
// be selected or left as-is. Such references are the end goal of resolution.
type ReferenceGraph struct {
	mu       sync.RWMutex
	refs     collections.Map[*globalref.Reference, *globalref.Reference]
	selector func(*globalref.Reference) bool
}

func NewReferenceGraph(selector func(*globalref.Reference) bool) *ReferenceGraph {
	return &ReferenceGraph{
		refs:     collections.NewMapFunc[*globalref.Reference, *globalref.Reference](globalReferenceUniqueKeyFunc),
		selector: selector,
	}
}

// SetReference records a source reference and the expression it points to.
func (s *ReferenceGraph) SetReference(sourceRef *globalref.Reference, expr hcl.Expression, exprModule addrs.ModuleInstance) {
	if s == nil {
		return
	}
	if sourceRef.LocalRef == nil {
		return
	}
	if expr == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// If the expression doesn't reference any variables, then it is a literal
	// value and we can skip traversal-based resolution.
	if len(expr.Variables()) == 0 {
		s.refs.Put(sourceRef, nil)
		return
	}

	exprTraversal, hclDiags := hcl.AbsTraversalForExpr(expr)
	if hclDiags.HasErrors() {
		// Not a single absolute traversal. Fall back to decomposing the
		// reference-preserving subset (splat/element/homogeneous conditional/…);
		// this records references laundered through a non-simple local or output.
		s.recordDecomposed(sourceRef, expr, exprModule)
		return
	}

	// parse the expression using exprModule as its container.
	exprRef, diags := globalref.ParseRef(exprModule, exprTraversal)
	if diags.HasErrors() {
		return
	}

	// First check if the expression already resolves through previously
	// recorded simple traversals, e.g. a local variable that points to another local variable.
	if resolved, ok := s.resolveReference(exprRef); ok {
		s.set(sourceRef, resolved)
		return
	}

	// Otherwise store the direct reference if it is a resource attribute reference.
	if s.selector != nil {
		if ok := s.selector(exprRef); ok {
			s.set(sourceRef, exprRef)
			return
		}
	}

	// If the expression is itself a simple pass-through to a constant (e.g. a
	// local or output whose recorded value resolves to a literal), propagate the
	// constant marker so this source is also recorded as resolving to a
	// constant. This keeps constant-ness flowing through the graph the same way
	// resource references do, so a hardcoded value laundered through one or more
	// intermediate locals/outputs is still resolvable as a definitive
	// non-reference rather than silently dropped.
	if s.resolvesToLiteral(exprRef, 0) {
		s.refs.Put(sourceRef, nil)
		return
	}
}

// recordDecomposed records a source reference whose expression is not a single
// absolute traversal, by decomposing the reference-preserving subset of
// expressions (splats, conditionals, allowlisted functions). It records the
// source only when its provenance is decidable: a single homogeneous resource
// reference (recorded as that reference) or a pure constant (recorded as nil).
// Heterogeneous, mixed reference/constant, or opaque expressions are left
// unrecorded so consumers treat them as unknown. Callers hold the write lock.
func (s *ReferenceGraph) recordDecomposed(sourceRef *globalref.Reference, expr hcl.Expression, exprModule addrs.ModuleInstance) {
	traversals, hasNonRef, ok := DecomposeTraversals(expr)
	if !ok {
		return
	}

	var resolved *globalref.Reference
	for _, tr := range traversals {
		ref, diags := globalref.ParseRef(exprModule, tr)
		if diags.HasErrors() {
			return
		}
		if res, rok := s.resolveReference(ref); rok {
			if resolved == nil {
				resolved = res
			} else if !equalConfigRef(resolved, res) {
				// Two distinct resources reachable -> undecidable.
				return
			}
		} else if s.resolvesToLiteral(ref, 0) {
			hasNonRef = true
		} else {
			// Unresolved and not a known constant -> unknown.
			return
		}
	}

	if hasNonRef {
		// A constant branch is present. Only definitive if there is no
		// competing resource reference (pure constant); otherwise undecidable.
		if resolved == nil {
			s.refs.Put(sourceRef, nil)
		}
		return
	}
	if resolved != nil {
		s.set(sourceRef, resolved)
	}
}

// equalConfigRef reports whether two references point at the same resource
// attribute at config-address granularity (instance keys normalized away).
func equalConfigRef(a, b *globalref.Reference) bool {
	if a == nil || b == nil {
		return false
	}
	aa, ok := a.ResourceAttr()
	if !ok {
		return false
	}
	bb, ok := b.ResourceAttr()
	if !ok {
		return false
	}
	if !aa.Resource.ConfigResource().Equal(bb.Resource.ConfigResource()) {
		return false
	}
	return aa.Attr.Equals(bb.Attr)
}

func (s *ReferenceGraph) ResolveReference(ref *globalref.Reference) (*globalref.Reference, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolveReference(ref)
}

// ResolvesToLiteral reports whether the given reference resolves (transitively)
// to a constant/literal value rather than a resource attribute — for example a
// local or module output whose expression is a hardcoded string. Such a
// reference is definitively not a config reference to another resource, so
// callers can treat it as a hard non-match rather than an unknown.
//
// Constant-ness is propagated through simple pass-through chains by
// SetReference (the same way resource references are), so a constant laundered
// through one or more intermediate locals/outputs is still detected. A value
// only surfaces as "not a literal" here when it was never recorded — e.g. a
// non-simple expression (a function call, splat, or conditional) that
// SetReference could not reduce to a single absolute traversal; callers should
// treat that as unknown.
func (s *ReferenceGraph) ResolvesToLiteral(ref *globalref.Reference) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolvesToLiteral(ref, 0)
}

func (s *ReferenceGraph) resolvesToLiteral(ref *globalref.Reference, depth int) bool {
	if s == nil || ref == nil || ref.LocalRef == nil {
		return false
	}
	if depth > 100 {
		return false
	}
	// A resource attribute reference is a terminal, not a literal.
	if s.selector != nil && s.selector(ref) {
		return false
	}
	resolved, ok := s.refs.GetOk(ref)
	if !ok {
		return false
	}
	if resolved == nil || resolved.LocalRef == nil {
		// Recorded explicitly as resolving to a literal (no resolvable target).
		return true
	}
	return s.resolvesToLiteral(resolved, depth+1)
}

// resolveReference tries to resolve the given reference. If the reference matches the selector,
// then resolution is done. Otherwise, we look up the local store.
func (s *ReferenceGraph) resolveReference(ref *globalref.Reference) (*globalref.Reference, bool) {
	if s == nil {
		return nil, false
	}
	if ref == nil || ref.LocalRef == nil {
		return nil, false
	}

	if s.selector != nil {
		if ok := s.selector(ref); ok {
			return ref, true
		}
	}

	return s.get(ref)
}

func (s *ReferenceGraph) set(sourceRef *globalref.Reference, resolvedRef *globalref.Reference) {
	if sourceRef == nil || sourceRef.LocalRef == nil {
		return
	}
	if resolvedRef != nil && resolvedRef.LocalRef == nil {
		return
	}

	s.refs.Put(sourceRef, resolvedRef)
}

func (s *ReferenceGraph) get(ref *globalref.Reference) (*globalref.Reference, bool) {
	if ref == nil {
		return nil, false
	}

	resolved, ok := s.refs.GetOk(ref)
	if !ok || resolved == nil || resolved.LocalRef == nil {
		return nil, false
	}

	if s.selector != nil {
		if ok := s.selector(resolved); ok {
			return resolved, true
		}
	}

	return nil, false
}

type globalReferenceUniqueKey string

func (globalReferenceUniqueKey) IsUniqueKey(*globalref.Reference) {}

func globalReferenceUniqueKeyFunc(ref *globalref.Reference) collections.UniqueKey[*globalref.Reference] {
	if ref == nil || ref.ContainerAddr == nil || ref.LocalRef == nil {
		return globalReferenceUniqueKey("<invalid>")
	}
	return globalReferenceUniqueKey(fmt.Sprintf("%s(%T)%s", containerConfigString(ref.ContainerAddr), ref.LocalRef.Subject, ref.LocalRef.DisplayString()))
}

// containerConfigString returns a config-address-granular string for a
// reference container, normalizing module-instance keys (from `count`/`for_each`
// expansion) away. Recorders such as node_output and node_module_variable key
// references by their fully-expanded module *instance* (e.g. module.buckets[0]),
// while a connector reference into that module is resolved at config-address
// granularity (module.buckets, index stripped). Keying both on the config module
// lets provenance chain through references that pass through expanded modules.
func containerConfigString(container addrs.Targetable) string {
	if mi, ok := container.(addrs.ModuleInstance); ok {
		return mi.Module().String()
	}
	return container.String()
}
