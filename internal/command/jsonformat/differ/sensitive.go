package differ

import "github.com/hashicorp/terraform/internal/command/jsonformat/change"

func (v Value) checkForSensitive() (change.Change, bool) {
	beforeSensitive := v.isBeforeSensitive()
	afterSensitive := v.isAfterSensitive()

	if !beforeSensitive && !afterSensitive {
		return change.Change{}, false
	}

	return v.asChange(change.Sensitive(v.Before, v.After, beforeSensitive, afterSensitive)), true
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
