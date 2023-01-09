package jsonprovider

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

type Schema struct {
	Version uint64 `json:"version"`
	Block   *Block `json:"block,omitempty"`
}

// marshalSchema is a convenience wrapper around mashalBlock. Schema version
// should be set by the caller.
func marshalSchema(block *configschema.Block) *Schema {
	if block == nil {
		return &Schema{}
	}

	var ret Schema
	ret.Block = marshalBlock(block)

	return &ret
}

func marshalSchemas(blocks map[string]*configschema.Block, rVersions map[string]uint64) map[string]*Schema {
	if blocks == nil {
		return map[string]*Schema{}
	}
	ret := make(map[string]*Schema, len(blocks))
	for k, v := range blocks {
		ret[k] = marshalSchema(v)
		version, ok := rVersions[k]
		if ok {
			ret[k].Version = version
		}
	}
	return ret
}
