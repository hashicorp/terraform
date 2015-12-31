package azurerm

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// makeHashFunction is a helper function which; given a slice of field names
// and a list of the names returns a schema.SchemaSetFunc
func makeHashFunction(simpleFields []string, listFields []string) schema.SchemaSetFunc {
	return func(v interface{}) int {
		m := v.(map[string]interface{})
		s := ""

		// first; fetch the simple fields:
		for _, field := range simpleFields {
			if val, ok := m[field]; ok {
				// NOTE: some fields may hold integers or booleans:
				s = s + fmt.Sprintf("%v", val)
			}
		}

		// then; fetch the list fields:
		for _, field := range listFields {
			if val, ok := m[field]; ok {
				for _, item := range val.([]interface{}) {
					s = s + fmt.Sprintf("%v", item)
				}
			}
		}

		return hashcode.String(s)
	}
}
