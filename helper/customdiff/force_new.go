package customdiff

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// ForceNewIf returns a CustomizeDiffFunc that flags the given key as
// requiring a new resource if the given condition function returns true.
//
// The return value of the condition function is ignored if the old and new
// values of the field compare equal, since no attribute diff is generated in
// that case.
func ForceNewIf(key string, f ResourceConditionFunc) schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		if f(d, meta) {
			d.ForceNew(key)
		}
		return nil
	}
}

// ForceNewIfChange returns a CustomizeDiffFunc that flags the given key as
// requiring a new resource if the given condition function returns true.
//
// The return value of the condition function is ignored if the old and new
// values compare equal, since no attribute diff is generated in that case.
//
// This function is similar to ForceNewIf but provides the condition function
// only the old and new values of the given key, which leads to more compact
// and explicit code in the common case where the decision can be made with
// only the specific field value.
func ForceNewIfChange(key string, f ValueChangeConditionFunc) schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		old, new := d.GetChange(key)
		if f(old, new, meta) {
			d.ForceNew(key)
		}
		return nil
	}
}
