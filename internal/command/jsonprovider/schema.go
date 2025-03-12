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
	if schema.Body == nil {
		return &Schema{}
	}

	var ret Schema
	ret.Block = marshalBlock(schema.Body)
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

type IdentitySchema struct {
	Version    uint64                        `json:"version"`
	Attributes map[string]*IdentityAttribute `json:"attributes,omitempty"`
}

func marshalIdentitySchema(schema providers.Schema) *IdentitySchema {
	var ret IdentitySchema
	ret.Version = uint64(schema.IdentityVersion)
	ret.Attributes = make(map[string]*IdentityAttribute, len(schema.Identity.Attributes))

	for k, v := range schema.Identity.Attributes {
		ret.Attributes[k] = marshalIdentityAttribute(v)
	}

	return &ret
}

func marshalIdentitySchemas(schemas map[string]providers.Schema) map[string]*IdentitySchema {
	if schemas == nil {
		return map[string]*IdentitySchema{}
	}

	ret := make(map[string]*IdentitySchema, len(schemas))
	for k, v := range schemas {
		if v.Identity != nil {
			ret[k] = marshalIdentitySchema(v)
		}
	}

	return ret
}
