package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/computed/renderers"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/plans"
)

type CreateSensitiveRenderer func(computed.Diff, bool, bool) computed.DiffRenderer

func checkForSensitiveType(change structured.Change, ctype cty.Type) (computed.Diff, bool) {
	return checkForSensitive(change, renderers.Sensitive, func(value structured.Change) computed.Diff {
		return ComputeDiffForType(value, ctype)
	})
}

func checkForSensitiveNestedAttribute(change structured.Change, attribute *jsonprovider.NestedType) (computed.Diff, bool) {
	return checkForSensitive(change, renderers.Sensitive, func(value structured.Change) computed.Diff {
		return computeDiffForNestedAttribute(value, attribute)
	})
}

func checkForSensitiveBlock(change structured.Change, block *jsonprovider.Block) (computed.Diff, bool) {
	return checkForSensitive(change, renderers.SensitiveBlock, func(value structured.Change) computed.Diff {
		return ComputeDiffForBlock(value, block)
	})
}

func checkForSensitive(change structured.Change, create CreateSensitiveRenderer, computedDiff func(value structured.Change) computed.Diff) (computed.Diff, bool) {
	beforeSensitive := change.IsBeforeSensitive()
	afterSensitive := change.IsAfterSensitive()

	if !beforeSensitive && !afterSensitive {
		return computed.Diff{}, false
	}

	// We are still going to give the change the contents of the actual change.
	// So we create a new Change with everything matching the current value,
	// except for the sensitivity.
	//
	// The change can choose what to do with this information, in most cases
	// it will just be ignored in favour of printing `(sensitive value)`.

	value := structured.Change{
		BeforeExplicit:     change.BeforeExplicit,
		AfterExplicit:      change.AfterExplicit,
		Before:             change.Before,
		After:              change.After,
		Unknown:            change.Unknown,
		BeforeSensitive:    false,
		AfterSensitive:     false,
		ReplacePaths:       change.ReplacePaths,
		RelevantAttributes: change.RelevantAttributes,
	}

	inner := computedDiff(value)

	action := inner.Action

	sensitiveStatusChanged := beforeSensitive != afterSensitive

	// nullNoOp is a stronger NoOp, where not only is there no change happening
	// but the before and after values are not explicitly set and are both
	// null. This will override even the sensitive state changing.
	nullNoOp := change.Before == nil && !change.BeforeExplicit && change.After == nil && !change.AfterExplicit

	if action == plans.NoOp && sensitiveStatusChanged && !nullNoOp {
		// Let's override this, since it means the sensitive status has changed
		// rather than the actual content of the value.
		action = plans.Update
	}

	return computed.NewDiff(create(inner, beforeSensitive, afterSensitive), action, change.ReplacePaths.Matches()), true
}
