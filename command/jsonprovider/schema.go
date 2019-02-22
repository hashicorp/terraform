package jsonprovider

import (
	"github.com/hashicorp/terraform/configs/configschema"
)

type schema struct {
	Version uint64 `json:"version,omitempty"`
	Block   *block `json:"block,omitempty"`
}

func marshalSchema(block *configschema.Block) *schema {
	if block == nil {
		return &schema{}
	}

	var ret schema
	ret.Block = marshalBlock(block)

	return &ret
}

func marshalSchemas(blocks map[string]*configschema.Block, rVersions map[string]uint64) map[string]*schema {
	if blocks == nil {
		return map[string]*schema{}
	}
	ret := make(map[string]*schema, len(blocks))
	for k, v := range blocks {
		ret[k] = marshalSchema(v)
		version, ok := rVersions[k]
		if ok {
			ret[k].Version = version
		}
	}
	return ret
}
