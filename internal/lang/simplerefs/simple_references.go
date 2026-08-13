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
}

func (s *ReferenceGraph) ResolveReference(ref *globalref.Reference) (*globalref.Reference, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.resolveReference(ref)
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
	return globalReferenceUniqueKey(fmt.Sprintf("%s(%T)%s", ref.ContainerAddr.String(), ref.LocalRef.Subject, ref.LocalRef.DisplayString()))
}
