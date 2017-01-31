package circonus

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

// suppressFuncs takes a list of functions and runs them in serial until the
// first functor returns that a result suggesting there is a diff that can't be
// ignored.
func suppressFuncs(fns ...func(k, old, new string, d *schema.ResourceData) bool) func(k, old, new string, d *schema.ResourceData) bool {
	return func(k, old, new string, d *schema.ResourceData) bool {
		for _, fn := range fns {
			if fn(k, old, new, d) {
				return true
			}
		}
		return false
	}
}

func suppressWhitespace(v interface{}) string {
	return strings.TrimSpace(v.(string))
}
