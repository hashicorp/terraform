package consul

import "github.com/hashicorp/terraform/helper/schema"

type attrWriter interface {
	BackingType() string

	SetBool(schemaAttr, bool) error
	SetFloat64(schemaAttr, float64) error
	SetList(schemaAttr, []interface{}) error
	SetMap(schemaAttr, map[string]interface{}) error
	SetSet(schemaAttr, *schema.Set) error
	SetString(schemaAttr, string) error
}
