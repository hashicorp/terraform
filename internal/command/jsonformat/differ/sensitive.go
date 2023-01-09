package differ

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/command/jsonformat/change"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
)

func (v Value) checkForSensitiveType(ctype cty.Type) (change.Change, bool) {
	return v.checkForSensitive(func(value Value) change.Change {
		return value.computeChangeForType(ctype)
	})
}

func (v Value) checkForSensitiveNestedAttribute(attribute *jsonprovider.NestedType) (change.Change, bool) {
	return v.checkForSensitive(func(value Value) change.Change {
		return value.computeChangeForNestedAttribute(attribute)
	})
}

func (v Value) checkForSensitiveBlock(block *jsonprovider.Block) (change.Change, bool) {
	return v.checkForSensitive(func(value Value) change.Change {
		return value.ComputeChangeForBlock(block)
	})
}

func (v Value) checkForSensitive(computeChange func(value Value) change.Change) (change.Change, bool) {
	beforeSensitive := v.isBeforeSensitive()
	afterSensitive := v.isAfterSensitive()

	if !beforeSensitive && !afterSensitive {
		return change.Change{}, false
	}

	// We are still going to give the change the contents of the actual change.
	// So we create a new Value with everything matching the current value,
	// except for the sensitivity.
	//
	// The change can choose what to do with this information, in most cases
	// it will just be ignored in favour of printing `(sensitive value)`.

	value := Value{
		BeforeExplicit:  v.BeforeExplicit,
		AfterExplicit:   v.AfterExplicit,
		Before:          v.Before,
		After:           v.After,
		Unknown:         v.Unknown,
		BeforeSensitive: false,
		AfterSensitive:  false,
		ReplacePaths:    v.ReplacePaths,
	}

	inner := computeChange(value)

	return change.New(change.Sensitive(inner, beforeSensitive, afterSensitive), inner.Action(), v.replacePath()), true
}

func (v Value) isBeforeSensitive() bool {
	if sensitive, ok := v.BeforeSensitive.(bool); ok {
		return sensitive
	}
	return false
}

func (v Value) isAfterSensitive() bool {
	if sensitive, ok := v.AfterSensitive.(bool); ok {
		return sensitive
	}
	return false
}
