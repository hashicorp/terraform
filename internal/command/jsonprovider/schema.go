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

type ActionSchema struct {
	ConfigSchema *Block `json:"block,omitempty"`

	// One of the following must be set
	Unlinked  *UnlinkedAction  `json:"unlinked,omitempty"`
	Linked    *LinkedAction    `json:"linked,omitempty"`
	Lifecycle *LifecycleAction `json:"lifecycle,omitempty"`
}

type UnlinkedAction struct{}
type LinkedAction struct {
	LinkedResources []LinkedResourceSchema `json:"linked_resources,omitempty"`
}
type LinkedResourceSchema struct {
	TypeName string `json:"type_name"`
}

type LifecycleAction struct {
	LinkedResource LinkedResourceSchema `json:"linked_resource"`
	ExecutionOrder string               `json:"execution_order"`
}

func marshalActionSchemas(schemas map[string]providers.ActionSchema) map[string]*ActionSchema {
	ret := make(map[string]*ActionSchema, len(schemas))
	for name, schema := range schemas {
		ret[name] = marshalActionSchema(schema)
	}
	return ret
}

func marshalActionSchema(schema providers.ActionSchema) *ActionSchema {
	ret := &ActionSchema{
		ConfigSchema: marshalBlock(schema.ConfigSchema),
	}

	if schema.Unlinked != nil {
		ret.Unlinked = &UnlinkedAction{}
	} else if schema.Linked != nil {
		linkedResources := []LinkedResourceSchema{}
		for _, linkedResource := range schema.Linked.LinkedResources {
			linkedResources = append(linkedResources, LinkedResourceSchema{
				TypeName: linkedResource.TypeName,
			})
		}
		ret.Linked = &LinkedAction{
			LinkedResources: linkedResources,
		}
	} else if schema.Lifecycle != nil {
		ret.Lifecycle = &LifecycleAction{
			LinkedResource: LinkedResourceSchema{
				TypeName: schema.Lifecycle.LinkedResource.TypeName,
			},
			ExecutionOrder: string(schema.Lifecycle.Executes),
		}
	}

	return ret
}
