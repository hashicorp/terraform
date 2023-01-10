package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"

	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (change Change) checkForUnknownType(ctype cty.Type) (computed.Diff, bool) {
	return change.checkForUnknown(func(value Change) computed.Diff {
		return value.computeDiffForType(ctype)
	})
}
func (change Change) checkForUnknownNestedAttribute(attribute *jsonprovider.NestedType) (computed.Diff, bool) {
	return change.checkForUnknown(func(value Change) computed.Diff {
		return value.computeDiffForNestedAttribute(attribute)
	})
}

func (change Change) checkForUnknownBlock(block *jsonprovider.Block) (computed.Diff, bool) {
	return change.checkForUnknown(func(value Change) computed.Diff {
		return value.ComputeDiffForBlock(block)
	})
}

func (change Change) checkForUnknown(computeDiff func(value Change) computed.Diff) (computed.Diff, bool) {
	unknown := change.isUnknown()

	if !unknown {
		return computed.Diff{}, false
	}

	// No matter what we do here, we want to treat the after value as explicit.
	// This is because it is going to be null in the value, and we don't want
	// the functions in this package to assume this means it has been deleted.
	change.AfterExplicit = true

	if change.Before == nil {
		return change.asDiff(renderers.Unknown(computed.Diff{})), true
	}

	// If we get here, then we have a before value. We're going to model a
	// delete operation and our renderer later can render the overall change
	// accurately.

	beforeValue := Change{
		Before:          change.Before,
		BeforeSensitive: change.BeforeSensitive,
	}
	return change.asDiff(renderers.Unknown(computeDiff(beforeValue))), true
}

func (change Change) isUnknown() bool {
	if unknown, ok := change.Unknown.(bool); ok {
		return unknown
	}
	return false
}
