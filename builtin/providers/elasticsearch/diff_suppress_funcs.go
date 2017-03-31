package elasticsearch

import (
	"encoding/json"
	"reflect"

	"github.com/hashicorp/terraform/helper/schema"
)

func diffSuppressIndexTemplate(k, old, new string, d *schema.ResourceData) bool {
	var oo, no interface{}
	if err := json.Unmarshal([]byte(old), &oo); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(new), &no); err != nil {
		return false
	}

	if om, ok := oo.(map[string]interface{}); ok {
		normalizeIndexTemplate(om)
	}

	if nm, ok := no.(map[string]interface{}); ok {
		normalizeIndexTemplate(nm)
	}

	return reflect.DeepEqual(oo, no)
}
