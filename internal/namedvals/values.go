// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package namedvals

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Values is a type we use internally to track values that were already
// resolved.
//
// This container encapsulates the problem of dealing with placeholder values
// in modules whose instance addresses are not yet fully known due to unknown
// values in count or for_each. Callers can register values in module paths
// of different levels of "known-ness" and then when queried the result will
// be the value from the most specific registration seen so far.
//
// Values is not concurrency-safe, because we expect to use it only indirectly
// through the public-facing [State] type and it should be the one to ensure
// concurrency safety.
type values[LocalType namedValueAddr, AbsType namedValueAddr] struct {
	// exact are values for objects in fully-qualified module instances.
	exact addrs.Map[AbsType, cty.Value]

	// placeholder are placeholder values to use for objects under an
	// unexpanded module prefix. These placeholders will contain known
	// values only in positions that are guaranteed to have equal values
	// across all module instances under a given prefix, thereby allowing
	// similar placeholder evaluation for other objects downstream of
	// the partially-evaluated ones.
	//
	// This one is more awkward because there can be conflicting placeholders
	// for the same object at different levels of specificity, and so we'll
	// need to scan over elements to find ones that match the most. To slightly
	// optimize that we have the values bucketed by the static module address
	// they belong to; maybe we'll optimize this more later if it seems like
	// a bottleneck, but we're assuming a relatively small number of declared
	// objects of each type in each module.
	placeholder addrs.Map[addrs.Module, addrs.Map[addrs.InPartialExpandedModule[LocalType], cty.Value]]
}

type inputVariableValues = values[addrs.InputVariable, addrs.AbsInputVariableInstance]
type localValues = values[addrs.LocalValue, addrs.AbsLocalValue]
type outputValues = values[addrs.OutputValue, addrs.AbsOutputValue]

// namedValueAddr describes the address behaviors we need for address types
// we'll use with type [values] as defined above.
type namedValueAddr interface {
	addrs.UniqueKeyer
	fmt.Stringer
}

func newValues[LocalType namedValueAddr, AbsType namedValueAddr]() values[LocalType, AbsType] {
	return values[LocalType, AbsType]{
		exact:       addrs.MakeMap[AbsType, cty.Value](),
		placeholder: addrs.MakeMap[addrs.Module, addrs.Map[addrs.InPartialExpandedModule[LocalType], cty.Value]](),
	}
}

func (v *values[LocalType, AbsType]) SetExactResult(addr AbsType, val cty.Value) {
	if v.exact.Has(addr) {
		// This is always a bug in the caller, because Terraform Core should
		// use its graph to ensure that each value gets set exactly once and
		// the values get registered in the correct order.
		panic(fmt.Sprintf("value for %s was already set by an earlier caller", addr))
	}
	v.exact.Put(addr, val)
}

func (v *values[LocalType, AbsType]) HasExactResult(addr AbsType) bool {
	return v.exact.Has(addr)
}

func (v *values[LocalType, AbsType]) GetExactResult(addr AbsType) cty.Value {
	// TODO: Do we need to handle placeholder results in here too? Seems like
	// callers should not end up holding AbsType addresses if they are dealing
	// with unexpanded objects because they wouldn't have instance keys to
	// fill in to the address, so assuming not for now.
	if !v.exact.Has(addr) {
		// This is always a bug in the caller, because Terraform Core should
		// use its graph to ensure that each value gets set exactly once and
		// the values get registered in the correct order.
		panic(fmt.Sprintf("value for %s was requested before it was provided", addr))
	}
	return v.exact.Get(addr)
}

func (v *values[LocalType, AbsType]) GetExactResults() addrs.Map[AbsType, cty.Value] {
	return v.exact
}

func (v *values[LocalType, AbsType]) SetPlaceholderResult(addr addrs.InPartialExpandedModule[LocalType], val cty.Value) {
	modAddr := addr.Module.Module()
	if !v.placeholder.Has(modAddr) {
		v.placeholder.Put(modAddr, addrs.MakeMap[addrs.InPartialExpandedModule[LocalType], cty.Value]())
	}
	placeholders := v.placeholder.Get(modAddr)
	if placeholders.Has(addr) {
		// This is always a bug in the caller, because Terraform Core should
		// use its graph to ensure that each value gets set exactly once and
		// the values get registered in the correct order.
		panic(fmt.Sprintf("placeholder value for %s was already set by an earlier caller", addr))
	}
	placeholders.Put(addr, val)
}

func (v *values[LocalType, AbsType]) GetPlaceholderResult(addr addrs.InPartialExpandedModule[LocalType]) cty.Value {
	modAddr := addr.Module.Module()
	if !v.placeholder.Has(modAddr) {
		return cty.DynamicVal
	}

	// We'll now search the placeholders for just the ones that include the
	// given address, and take the most specific of them, which is the one with
	// the longest known module prefix.
	var candidates []addrs.MapElem[addrs.InPartialExpandedModule[LocalType], cty.Value]
	for _, elem := range v.placeholder.Get(modAddr).Elems {
		if addrs.Equivalent(elem.Key.Local, addr.Local) && elem.Key.Module.MatchesPartial(addr.Module) {
			candidates = append(candidates, elem)
		}
	}

	best, ok := addrs.MostSpecific(candidates, func(a, b addrs.MapElem[addrs.InPartialExpandedModule[LocalType], cty.Value]) bool {
		return a.Key.Module.MatchesPartial(b.Key.Module)
	})
	if !ok {
		return cty.DynamicVal
	}
	return best.Value
}
