package marks

import (
	"sort"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

type resourceInstanceDerived struct {
	instanceAddr addrs.AbsResourceInstance
}

// MarkAsResourceInstanceValue returns a version of the given value that
// carries an additional mark recording that it originated from a particular
// given resource instance.
//
// We don't use such markings for all values derived from resource instances,
// but in some specialized situations we use this function and its companions
// ResourceInstancesDerivedFrom and HasResourceInstanceValueMarks as part of
// dynamic analysis to gather additional context to return in error messages.
func MarkAsResourceInstanceValue(v cty.Value, instanceAddr addrs.AbsResourceInstance) cty.Value {
	// We can't use an addrs.AbsResourceInstance _directly_ as a mark,
	// because it contains a slice, but we'll wrap it up inside a
	// pointer to a resourceInstanceDerived so cty can compare it by pointer
	// equality. In practice that means that no two marks produced by
	// this function will actually be equal, but that's okay because
	// we'll dedupe them on the way back out in ResourceInstancesDerivedFrom.
	// That defers the effort of deduping them to only if we actually end
	// up using the marks, at the expense of some usually-minor additional
	// overhead of potentially tracking the same address multiple times.
	return v.Mark(&resourceInstanceDerived{instanceAddr})
}

// HasResourceInstanceValueMarks returns true if the given value carries any
// marks that originated from a call to MarkAsResourceInstanceValue.
//
// This can be considerably cheaper than calling ResourceInstancesDerivedFrom
// and testing whether the result is empty, because it avoids the need for
// any deduping or sorting of the result, and for allocating a new slice to
// return the results in.
func HasResourceInstanceValueMarks(v cty.Value) bool {
	marks := v.Marks()
	for mark := range marks {
		if _, ok := mark.(*resourceInstanceDerived); ok {
			return true
		}
	}
	return false
}

// ResourceInstancesDerivedFrom processes a value that was possibly previously
// marked using MarkAsResourceInstanceValue, and if so returns a set
// (presented as a slice) of all of the resource instance addresses the
// value has been marked with.
//
// The result contains only one element for each distinct instance address,
// and is in our usual sorting order for resource instance addresses.
//
// If the given value wasn't previously marked by MarkAsResourceInstanceValue
// or derived from a value that was, the result will be empty.
func ResourceInstancesDerivedFrom(v cty.Value) []addrs.AbsResourceInstance {
	marks := v.Marks()
	if len(marks) == 0 {
		return nil // easy case for totally-unmarked values
	}

	// We need to do some extra work here to dedupe the addresses. We
	// intentionally defer this to here, rather than doing it at marking
	// time, so that we only pay the deduping overhead of doing this if we
	// are actually going to use the result.
	addrsUniq := make(map[string]addrs.AbsResourceInstance, len(marks))
	for mark := range marks {
		if mark, ok := mark.(*resourceInstanceDerived); ok {
			// Here we assume that the string representation of an instance
			// address is sufficiently unique, which it is in this case
			// because we're only comparing instances to other instances.
			// (This would not be valid if possibly comparing a mixture of
			// resources and resource instances, due to ambiguity.)
			addrsUniq[mark.instanceAddr.String()] = mark.instanceAddr
		}
	}

	if len(addrsUniq) == 0 {
		// Could get here if the value only has marks _other than_
		// resourceInstanceDerived ones.
		return nil
	}

	addrs := make([]addrs.AbsResourceInstance, 0, len(addrsUniq))
	for _, addr := range addrsUniq {
		addrs = append(addrs, addr)
	}
	sort.SliceStable(addrs, func(i, j int) bool {
		return addrs[i].Less(addrs[j])
	})
	return addrs
}
