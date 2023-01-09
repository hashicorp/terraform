package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) checkForComputedType(ctype cty.Type) (change.Change, bool) {
	return v.checkForComputed(func(value Value) change.Change {
		return value.computeChangeForType(ctype)
	})
}
func (v Value) checkForComputedNestedAttribute(attribute *jsonprovider.NestedType) (change.Change, bool) {
	return v.checkForComputed(func(value Value) change.Change {
		return value.computeChangeForNestedAttribute(attribute)
	})
}

func (v Value) checkForComputedBlock(block *jsonprovider.Block) (change.Change, bool) {
	return v.checkForComputed(func(value Value) change.Change {
		return value.ComputeChangeForBlock(block)
	})
}

func (v Value) checkForComputed(computeChange func(value Value) change.Change) (change.Change, bool) {
	unknown := v.isUnknown()

	if !unknown {
		return change.Change{}, false
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	v.AfterExplicit = true

	if v.Before == nil {
		return v.asChange(change.Computed(change.Change{})), true
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.

	beforeValue := Value{
		Before:          v.Before,
		BeforeSensitive: v.BeforeSensitive,
	}
	return v.asChange(change.Computed(computeChange(beforeValue))), true
}

func (v Value) isUnknown() bool {
	if unknown, ok := v.Unknown.(bool); ok {
		return unknown
	}
	return false
}
