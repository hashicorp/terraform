package schema

import (
	"log"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// ResourceConfigGetter is a ResourceData-like interface for
// ResourceValidateFunc implementors. It is different from ResourceData in that
// it requires callers to check a second "known" return value before
// interacting with any value.
type ResourceConfigGetter interface {
	// GetIfKnown is the only function available to ResourceValidateFuncs.
	//
	// If known is true, value is guaranteed to be the type specified by the Schema
	// for this field.
	//
	// If known is false, the value will always be nil, and the Config cannot be
	// assumed to have a value one way or the other.
	//
	// It is the caller's responsibility to check known before using value, and
	// to treat unknown values appropriately in validations.
	GetIfKnown(key string) (value interface{}, known bool)
}

// ResourceConfigData wraps a Config and implements ResourceConfigGetter
type ResourceConfigData struct {
	reader *ConfigFieldReader
}

// NewResourceConfigData yields a ResourceConfigData for the provided config
// and schema.
func NewResourceConfigData(c *terraform.ResourceConfig, s map[string]*Schema) *ResourceConfigData {
	return &ResourceConfigData{
		reader: &ConfigFieldReader{Config: c, Schema: s},
	}
}

// GetIfKnown implements ResourceConfigGetter for ResourceConfigData
func (r *ResourceConfigData) GetIfKnown(key string) (interface{}, bool) {
	addr := strings.Split(key, ".")
	result, err := r.reader.ReadField(addr)
	if err != nil {
		log.Printf("[ERROR] Error during ResourceConfigData.Get: %s", err)
		return nil, false
	}

	// Computed results are simply unknown
	if result.Computed {
		return nil, false
	}

	// If the result doesn't exist, then we set the value to the zero value
	var schema *Schema
	if schemaL := addrToSchema(addr, r.reader.Schema); len(schemaL) > 0 {
		schema = schemaL[len(schemaL)-1]
	}

	if result.Value == nil && schema != nil {
		result.Value = result.ValueOrZero(schema)
	}

	return result.Value, true
}
