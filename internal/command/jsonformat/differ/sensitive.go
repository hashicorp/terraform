package differ

import "github.com/hashicorp/terraform/internal/command/jsonformat/change"

func (v Value) checkForSensitive(changeType interface{}) (change.Change, bool) {
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

	inner := value.ComputeChange(changeType)

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
