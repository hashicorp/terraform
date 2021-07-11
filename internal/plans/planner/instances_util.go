package planner

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// singletonInstances is a pre-allocated result for an InstanceKeys method
// that wants to signal only a single instance with no key.
//
// Don't change the contents of this map.
var singletonInstances = map[addrs.InstanceKey]struct{}{
	addrs.NoKey: {},
}

func resolveInstanceRepetition(ctx context.Context, p *planner, forEach hcl.Expression, count hcl.Expression, scope *lang.Scope) cty.Value {
	switch {
	case forEach != nil:
		eachVal, diags := scope.EvalExpr(forEach, cty.DynamicPseudoType)

		// TODO: All of the usual validation rules for for_each, like that it
		// ought to be a map/object/set-of-strings, it can't be unknown, etc.

		p.AddDiagnostics(diags)
		if diags.HasErrors() {
			return cty.DynamicVal
		}

		// We'll normalize to always return an object value, for caller ease.
		if eachVal.Type().IsSetType() {
			attrs := make(map[string]cty.Value)
			for it := eachVal.ElementIterator(); it.Next(); {
				_, v := it.Element()
				attrs[v.AsString()] = v
			}
			return cty.ObjectVal(attrs)
		} else {
			return cty.ObjectVal(eachVal.AsValueMap())
		}

	case count != nil:
		// TODO: Implement
		return cty.NilVal
	default:
		// If neither repetition argument is set, we have only one instance
		// without any key.
		return cty.NilVal
	}
}

func instanceKeysForRepetition(proxyVal cty.Value) map[addrs.InstanceKey]struct{} {
	// The expansion request uses a subset of cty.Value values to signal
	// the different situations:
	// - Unknown value means that expansion failed and it emitted an error diagnostic
	// - cty.NilVal means it's a singleton instance, with no key.
	// - A number value, always in range for an int, represents a "count" result
	// - An object value represents a "for_each" result

	switch {
	case !proxyVal.IsKnown():
		return nil // halt further processing if expansion failed
	case proxyVal == cty.NilVal:
		return singletonInstances
	case proxyVal.Type() == cty.Number:
		var n int
		err := gocty.FromCtyValue(proxyVal, &n)
		if err != nil {
			panic(fmt.Sprintf("invalid expansion result proxy value %#v: %s", proxyVal, err))
		}
		ret := make(map[addrs.InstanceKey]struct{}, n)
		for i := 0; i < n; i++ {
			ret[addrs.IntKey(n)] = struct{}{}
		}
		return ret
	case proxyVal.Type().IsObjectType():
		atys := proxyVal.Type().AttributeTypes()
		ret := make(map[addrs.InstanceKey]struct{}, len(atys))
		for k := range atys {
			ret[addrs.StringKey(k)] = struct{}{}
		}
		return ret
	default:
		panic(fmt.Sprintf("invalid expansion result proxy value %#v", proxyVal))
	}
}

func eachValueForInstance(proxyVal cty.Value, key addrs.StringKey) cty.Value {
	if !proxyVal.Type().IsObjectType() {
		// Not a for_each resource then, so no "EachValue"
		return cty.NilVal
	}
	if !proxyVal.Type().HasAttribute(string(key)) {
		// There is no definition for this particular key.
		return cty.NilVal
	}
	return proxyVal.GetAttr(string(key))
}

func repetitionDataForInstance(ctx context.Context, key addrs.InstanceKey, eachValFunc func(context.Context, addrs.StringKey) cty.Value) instances.RepetitionData {
	switch key := key.(type) {
	case addrs.IntKey:
		return instances.RepetitionData{
			CountIndex: cty.NumberIntVal(int64(key)),
		}
	case addrs.StringKey:
		keyVal := cty.StringVal(string(key))

		// To get the "EachValue" we need the help of our associated
		// resource, so it can evaluate its own repetition settings.
		valVal := eachValFunc(ctx, key)

		return instances.RepetitionData{
			EachKey:   keyVal,
			EachValue: valVal,
		}
	default:
		return instances.RepetitionData{} // none
	}
}

func aggregateValueForInstances(ctx context.Context, proxyVal cty.Value, instVal func(context.Context, addrs.InstanceKey) cty.Value) cty.Value {
	switch {
	case !proxyVal.IsKnown():
		return cty.DynamicVal
	case proxyVal == cty.NilVal:
		return instVal(ctx, addrs.NoKey)
	case proxyVal.Type() == cty.Number:
		var n int
		err := gocty.FromCtyValue(proxyVal, &n)
		if err != nil {
			panic(fmt.Sprintf("invalid expansion result proxy value %#v: %s", proxyVal, err))
		}
		if n == 0 {
			return cty.EmptyTupleVal
		}
		elems := make([]cty.Value, 0, n)
		for i := 0; i < n; i++ {
			elems[i] = instVal(ctx, addrs.IntKey(i))
		}
		return cty.TupleVal(elems)
	case proxyVal.Type().IsObjectType():
		atys := proxyVal.Type().AttributeTypes()
		attrs := make(map[string]cty.Value, len(atys))
		for k := range atys {
			attrs[k] = instVal(ctx, addrs.StringKey(k))
		}
		return cty.ObjectVal(attrs)
	default:
		panic(fmt.Sprintf("invalid expansion result proxy value %#v", proxyVal))
	}

}
