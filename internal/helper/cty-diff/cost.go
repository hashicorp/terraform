package cty_diff

import (
	"github.com/zclconf/go-cty/cty"
)

// valueSize returns the number of nested cty values.
func valueSize(val cty.Value) float32 {
	ty := val.Type()

	switch {
	case !val.IsKnown() || val.IsNull() || ty.IsPrimitiveType():
		return 1
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType() || ty.IsMapType() || ty.IsObjectType():
		result := float32(1)
		for it := val.ElementIterator(); it.Next(); {
			_, value := it.Element()
			result += valueSize(value)
		}
		return result
	default:
		return 1
	}
}

// ValueDiffCost estimates the cost of calculating a diff between two cty objects using ValueDiff.
func ValueDiffCost(a, b cty.Value) float32 {
	if !ctyTypesEqual(a.Type(), b.Type()) {
		return 1
	} else {
		return valueSize(a) * valueSize(b)
	}
}
