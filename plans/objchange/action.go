package objchange

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/plans"
)

// ActionForChange determines which plans.Action value best describes a
// change from the value given in before to the value given in after.
//
// Because it has no context aside from the values, it can only return the
// basic actions NoOp, Create, Update, and Delete. Other codepaths with
// additional information might make this decision differently, such as by
// using the Replace action instead of the Update action where that makes
// sense.
//
// If the after value is unknown then the action can't be properly decided, and
// so ActionForChange will conservatively return either Create or Update
// depending on whether the before value is null. The before value must always
// be fully known; ActionForChange will panic if it contains any unknown values.
func ActionForChange(before, after cty.Value) plans.Action {
	switch {
	case !after.IsKnown():
		if before.IsNull() {
			return plans.Create
		}
		return plans.Update
	case after.IsNull() && before.IsNull():
		return plans.NoOp
	case after.IsNull() && !before.IsNull():
		return plans.Delete
	case before.IsNull() && !after.IsNull():
		return plans.Create
	case after.RawEquals(before):
		return plans.NoOp
	default:
		return plans.Update
	}
}
