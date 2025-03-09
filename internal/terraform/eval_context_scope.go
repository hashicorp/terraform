// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
)

// evalContextScope represents the scope that an [EvalContext] (or rather,
// an [EvalContextBuiltin] is associated with.
//
// This is a closed interface representing a sum type, with three possible
// variants:
//
//   - a nil value of this type represents a "global" evaluation context used
//     for graph nodes that aren't considered to belong to any specific module
//     instance. Some [EvalContext] methods are not appropriate for such a
//     context, and so will panic on a global evaluation context.
//   - [evalContextModuleInstance] is for an evaluation context used for
//     graph nodes that implement [GraphNodeModuleInstance], meaning that
//     they belong to a fully-expanded single module instance.
//   - [evalContextPartialExpandedModule] is for an evaluation context used for
//     graph nodes that implement [GraphNodeUnexpandedModule], meaning that
//     they belong to an unbounded set of possible module instances sharing
//     a common known prefix, in situations where a module call has an unknown
//     value for its count or for_each argument.
type evalContextScope interface {
	collections.UniqueKeyer[evalContextScope]

	// evalContextScopeModule returns the static module address of whatever
	// fully- or partially-expanded module instance address this scope is
	// associated with.
	//
	// A "global" evaluation context is a nil [evalContextScope], and so
	// this method will panic for that scope.
	evalContextScopeModule() addrs.Module

	String() string
}

// evalContextGlobal is the nil [evalContextScope] used to represent an
// [EvalContext] that isn't associated with any module at all.
var evalContextGlobal evalContextScope

// evalContextModuleInstance is an [evalContextScope] associated with a
// fully-expanded single module instance.
type evalContextModuleInstance struct {
	Addr addrs.ModuleInstance
}

func (s evalContextModuleInstance) evalContextScopeModule() addrs.Module {
	return s.Addr.Module()
}

func (s evalContextModuleInstance) String() string {
	return s.Addr.String()
}

func (s evalContextModuleInstance) UniqueKey() collections.UniqueKey[evalContextScope] {
	return evalContextScopeUniqueKey{
		k: s.Addr.UniqueKey(),
	}
}

// evalContextPartialExpandedModule is an [evalContextScope] associated with
// an unbounded set of possible module instances that share a common known
// address prefix.
type evalContextPartialExpandedModule struct {
	Addr addrs.PartialExpandedModule
}

func (s evalContextPartialExpandedModule) evalContextScopeModule() addrs.Module {
	return s.Addr.Module()
}

func (s evalContextPartialExpandedModule) String() string {
	return s.Addr.String()
}

func (s evalContextPartialExpandedModule) UniqueKey() collections.UniqueKey[evalContextScope] {
	return evalContextScopeUniqueKey{
		k: s.Addr.UniqueKey(),
	}
}

type evalContextScopeUniqueKey struct {
	k addrs.UniqueKey
}

// IsUniqueKey implements collections.UniqueKey.
func (evalContextScopeUniqueKey) IsUniqueKey(evalContextScope) {}
