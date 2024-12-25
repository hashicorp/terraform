// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
	"github.com/zclconf/go-cty/cty"
)

// ConfigSchemaToProto takes a *configschema.Block and converts it to a
// proto.Schema_Block for a grpc response.
func ConfigSchemaToProto(b *configschema.Block) *proto.Schema_Block {
	block := &proto.Schema_Block{
		Description:     b.Description,
		DescriptionKind: protoStringKind(b.DescriptionKind),
		Deprecated:      b.Deprecated,
	}

	for _, name := range sortedKeys(b.Attributes) {
		a := b.Attributes[name]

		attr := &proto.Schema_Attribute{
			Name:            name,
			Description:     a.Description,
			DescriptionKind: protoStringKind(a.DescriptionKind),
			Optional:        a.Optional,
			Computed:        a.Computed,
			Required:        a.Required,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
			WriteOnly:       a.WriteOnly,
		}

		if a.Type != cty.NilType {
			ty, err := json.Marshal(a.Type)
			if err != nil {
				panic(err)
			}
			attr.Type = ty
		}

		if a.NestedType != nil {
			attr.NestedType = configschemaObjectToProto(a.NestedType)
		}

		block.Attributes = append(block.Attributes, attr)
	}

	for _, name := range sortedKeys(b.BlockTypes) {
		b := b.BlockTypes[name]
		block.BlockTypes = append(block.BlockTypes, protoSchemaNestedBlock(name, b))
	}

	return block
}

func protoStringKind(k configschema.StringKind) proto.StringKind {
	switch k {
	default:
		return proto.StringKind_PLAIN
	case configschema.StringMarkdown:
		return proto.StringKind_MARKDOWN
	}
}

func protoSchemaNestedBlock(name string, b *configschema.NestedBlock) *proto.Schema_NestedBlock {
	var nesting proto.Schema_NestedBlock_NestingMode
	switch b.Nesting {
	case configschema.NestingSingle:
		nesting = proto.Schema_NestedBlock_SINGLE
	case configschema.NestingGroup:
		nesting = proto.Schema_NestedBlock_GROUP
	case configschema.NestingList:
		nesting = proto.Schema_NestedBlock_LIST
	case configschema.NestingSet:
		nesting = proto.Schema_NestedBlock_SET
	case configschema.NestingMap:
		nesting = proto.Schema_NestedBlock_MAP
	default:
		nesting = proto.Schema_NestedBlock_INVALID
	}
	return &proto.Schema_NestedBlock{
		TypeName: name,
		Block:    ConfigSchemaToProto(&b.Block),
		Nesting:  nesting,
		MinItems: int64(b.MinItems),
		MaxItems: int64(b.MaxItems),
	}
}

// ProtoToProviderSchema takes a proto.Schema and converts it to a providers.Schema.
func ProtoToProviderSchema(s *proto.Schema) providers.Schema {
	return providers.Schema{
		Version: s.Version,
		Block:   ProtoToConfigSchema(s.Block),
	}
}

// ProtoToConfigSchema takes the GetSchcema_Block from a grpc response and converts it
// to a terraform *configschema.Block.
func ProtoToConfigSchema(b *proto.Schema_Block) *configschema.Block {
	block := &configschema.Block{
		Attributes: make(map[string]*configschema.Attribute),
		BlockTypes: make(map[string]*configschema.NestedBlock),

		Description:     b.Description,
		DescriptionKind: schemaStringKind(b.DescriptionKind),
		Deprecated:      b.Deprecated,
	}

	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description:     a.Description,
			DescriptionKind: schemaStringKind(a.DescriptionKind),
			Required:        a.Required,
			Optional:        a.Optional,
			Computed:        a.Computed,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
			WriteOnly:       a.WriteOnly,
		}

		if a.Type != nil {
			if err := json.Unmarshal(a.Type, &attr.Type); err != nil {
				panic(err)
			}
		}

		if a.NestedType != nil {
			attr.NestedType = protoObjectToConfigSchema(a.NestedType)
		}

		block.Attributes[a.Name] = attr
	}

	for _, b := range b.BlockTypes {
		block.BlockTypes[b.TypeName] = schemaNestedBlock(b)
	}

	return block
}

func schemaStringKind(k proto.StringKind) configschema.StringKind {
	switch k {
	default:
		return configschema.StringPlain
	case proto.StringKind_MARKDOWN:
		return configschema.StringMarkdown
	}
}

func schemaNestedBlock(b *proto.Schema_NestedBlock) *configschema.NestedBlock {
	var nesting configschema.NestingMode
	switch b.Nesting {
	case proto.Schema_NestedBlock_SINGLE:
		nesting = configschema.NestingSingle
	case proto.Schema_NestedBlock_GROUP:
		nesting = configschema.NestingGroup
	case proto.Schema_NestedBlock_LIST:
		nesting = configschema.NestingList
	case proto.Schema_NestedBlock_MAP:
		nesting = configschema.NestingMap
	case proto.Schema_NestedBlock_SET:
		nesting = configschema.NestingSet
	default:
		// In all other cases we'll leave it as the zero value (invalid) and
		// let the caller validate it and deal with this.
	}

	nb := &configschema.NestedBlock{
		Nesting:  nesting,
		MinItems: int(b.MinItems),
		MaxItems: int(b.MaxItems),
	}

	nested := ProtoToConfigSchema(b.Block)
	nb.Block = *nested
	return nb
}

func protoObjectToConfigSchema(b *proto.Schema_Object) *configschema.Object {
	var nesting configschema.NestingMode
	switch b.Nesting {
	case proto.Schema_Object_SINGLE:
		nesting = configschema.NestingSingle
	case proto.Schema_Object_LIST:
		nesting = configschema.NestingList
	case proto.Schema_Object_MAP:
		nesting = configschema.NestingMap
	case proto.Schema_Object_SET:
		nesting = configschema.NestingSet
	default:
		// In all other cases we'll leave it as the zero value (invalid) and
		// let the caller validate it and deal with this.
	}

	object := &configschema.Object{
		Attributes: make(map[string]*configschema.Attribute),
		Nesting:    nesting,
	}

	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description:     a.Description,
			DescriptionKind: schemaStringKind(a.DescriptionKind),
			Required:        a.Required,
			Optional:        a.Optional,
			Computed:        a.Computed,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
			WriteOnly:       a.WriteOnly,
		}

		if a.Type != nil {
			if err := json.Unmarshal(a.Type, &attr.Type); err != nil {
				panic(err)
			}
		}

		if a.NestedType != nil {
			attr.NestedType = protoObjectToConfigSchema(a.NestedType)
		}

		object.Attributes[a.Name] = attr
	}

	return object
}

// sortedKeys returns the lexically sorted keys from the given map. This is
// used to make schema conversions are deterministic. This panics if map keys
// are not a string.
func sortedKeys(m interface{}) []string {
	v := reflect.ValueOf(m)
	keys := make([]string, v.Len())

	mapKeys := v.MapKeys()
	for i, k := range mapKeys {
		keys[i] = k.Interface().(string)
	}

	sort.Strings(keys)
	return keys
}

func configschemaObjectToProto(b *configschema.Object) *proto.Schema_Object {
	var nesting proto.Schema_Object_NestingMode
	switch b.Nesting {
	case configschema.NestingSingle:
		nesting = proto.Schema_Object_SINGLE
	case configschema.NestingList:
		nesting = proto.Schema_Object_LIST
	case configschema.NestingSet:
		nesting = proto.Schema_Object_SET
	case configschema.NestingMap:
		nesting = proto.Schema_Object_MAP
	default:
		nesting = proto.Schema_Object_INVALID
	}

	attributes := make([]*proto.Schema_Attribute, 0, len(b.Attributes))

	for _, name := range sortedKeys(b.Attributes) {
		a := b.Attributes[name]
		attr := &proto.Schema_Attribute{
			Name:            name,
			Description:     a.Description,
			DescriptionKind: protoStringKind(a.DescriptionKind),
			Optional:        a.Optional,
			Computed:        a.Computed,
			Required:        a.Required,
			Sensitive:       a.Sensitive,
			Deprecated:      a.Deprecated,
			WriteOnly:       a.WriteOnly,
		}

		if a.Type != cty.NilType {
			ty, err := json.Marshal(a.Type)
			if err != nil {
				panic(err)
			}
			attr.Type = ty
		}

		if a.NestedType != nil {
			attr.NestedType = configschemaObjectToProto(a.NestedType)
		}

		attributes = append(attributes, attr)
	}

	return &proto.Schema_Object{
		Attributes: attributes,
		Nesting:    nesting,
	}
}
