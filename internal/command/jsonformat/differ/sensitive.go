package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

type CreateSensitiveRenderer func(computed.Diff, bool, bool) computed.DiffRenderer

func (change Change) checkForSensitiveType(ctype cty.Type) (computed.Diff, bool) {
	return change.checkForSensitive(renderers.Sensitive, func(value Change) computed.Diff {
		return value.computeDiffForType(ctype)
	})
}

func (change Change) checkForSensitiveNestedAttribute(attribute *jsonprovider.NestedType) (computed.Diff, bool) {
	return change.checkForSensitive(renderers.Sensitive, func(value Change) computed.Diff {
		return value.computeDiffForNestedAttribute(attribute)
	})
}

func (change Change) checkForSensitiveBlock(block *jsonprovider.Block) (computed.Diff, bool) {
	return change.checkForSensitive(renderers.SensitiveBlock, func(value Change) computed.Diff {
		return value.ComputeDiffForBlock(block)
	})
}

func (change Change) checkForSensitive(create CreateSensitiveRenderer, computedDiff func(value Change) computed.Diff) (computed.Diff, bool) {
	beforeSensitive := change.isBeforeSensitive()
	afterSensitive := change.isAfterSensitive()

	if !beforeSensitive && !afterSensitive {
		return computed.Diff{}, false
	}

	// We are still going to give the change the contents of the actual change.
	// So we create a new Change with everything matching the current value,
	// except for the sensitivity.
	//
	// The change can choose what to do with this information, in most cases
	// it will just be ignored in favour of printing `(sensitive value)`.

	value := Change{
		BeforeExplicit:  change.BeforeExplicit,
		AfterExplicit:   change.AfterExplicit,
		Before:          change.Before,
		After:           change.After,
		Unknown:         change.Unknown,
		BeforeSensitive: false,
		AfterSensitive:  false,
		ReplacePaths:    change.ReplacePaths,
	}

	inner := computedDiff(value)

	action := inner.Action
	if action == plans.NoOp && beforeSensitive != afterSensitive {
		// Let's override this, since it means the sensitive status has changed
		// rather than the actual content of the value.
		action = plans.Update
	}

	return computed.NewDiff(create(inner, beforeSensitive, afterSensitive), action, change.ReplacePaths.ForcesReplacement()), true
}

func (change Change) isBeforeSensitive() bool {
	if sensitive, ok := change.BeforeSensitive.(bool); ok {
		return sensitive
	}
	return false
}

func (change Change) isAfterSensitive() bool {
	if sensitive, ok := change.AfterSensitive.(bool); ok {
		return sensitive
	}
	return false
}
