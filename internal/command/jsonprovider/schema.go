// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonprovider

import (
	"github.com/hashicorp/terraform/internal/providers"
)

type Schema struct {
	Version uint64 `json:"version"`
	Block   *Block `json:"block,omitempty"`
}

// marshalSchema is a convenience wrapper around mashalBlock. Schema version
// should be set by the caller.
func marshalSchema(schema providers.Schema) *Schema {
	if schema.Block == nil {
		return &Schema{}
	}

	var ret Schema
	ret.Block = marshalBlock(schema.Block)
	ret.Version = uint64(schema.Version)

	return &ret
}

func marshalSchemas(schemas map[string]providers.Schema) map[string]*Schema {
	if schemas == nil {
		return map[string]*Schema{}
	}
	ret := make(map[string]*Schema, len(schemas))
	for k, v := range schemas {
		ret[k] = marshalSchema(v)
	}
	return ret
}
