package customdiff

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// ComputedIf returns a CustomizeDiffFunc that sets the given key's new value
// as computed if the given condition function returns true.
func ComputedIf(key string, f ResourceConditionFunc) schema.CustomizeDiffFunc {
	return func(d *schema.ResourceDiff, meta interface{}) error {
		if f(d, meta) {
			d.SetNewComputed(key)
		}
		return nil
	}
}
