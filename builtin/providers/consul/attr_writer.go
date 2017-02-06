package consul

import "github.com/hashicorp/terraform/helper/schema"

type _AttrWriter interface {
	BackingType() string

	SetBool(_SchemaAttr, bool) error
	SetFloat64(_SchemaAttr, float64) error
	SetList(_SchemaAttr, []interface{}) error
	SetMap(_SchemaAttr, map[string]interface{}) error
	SetSet(_SchemaAttr, *schema.Set) error
	SetString(_SchemaAttr, string) error
}
