package azure

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// isFile is a helper method which stats for the existence of
// the provided filepath.
func isFile(v string) bool {
	_, err := os.Stat(v)
	return err == nil
}

// makeHashFunction is a helper function which; given a slice of field names
// and a list of the names
// returns a schema.SchemaSetFunc
func makeHashFunction(simpleFields []string, listFields []string) schema.SchemaSetFunc {
	return func(v interface{}) int {
		m := v.(map[string]interface{})
		s := ""

		// first; fetch the simple fields:
		for _, field := range simpleFields {
			if val, ok := m[field]; ok {
				// NOTE: some fields may hold integers  or booleans:
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

// readSet is a helper function which; given a *schema.Set and a slice with
// the name of the list fields which are treated specially and returns
// a slice of maps for each element of the set.
func readSet(v interface{}, listFieldNames []string) []map[string]interface{} {
	set := v.(*schema.Set)

	// check if set is empty:
	if set.Len() == 0 {
		return nil
	}

	// iterate through the elements, translate it into a map and add them all:
	res := []map[string]interface{}{}
	for _, elem := range set.List() {
		mp := map[string]interface{}{}

		// iterate through each element:
		for key, val := range elem.(map[string]interface{}) {
			// check if it is a list field:
			if in(key, listFieldNames) {
				list := []string{}

				for _, v := range val.([]interface{}) {
					list = append(list, v.(string))
				}

				mp[key] = list
			} else {
				// else, it's a simple field (either string or int),
				// so we just add it as-is:
				mp[key] = val
			}
		}
	}

	return res
}

// in is a helper function which determines whether or not a given string
// is present in the given slice of strings.
func in(str string, strs []string) bool {
	if strs == nil || len(strs) == 0 {
		return false
	}

	for _, s := range strs {
		if str == s {
			return true
		}
	}

	return false
}
