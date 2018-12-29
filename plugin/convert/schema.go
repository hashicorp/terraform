package convert

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform/configs/configschema"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/hashicorp/terraform/providers"
)

// ConfigSchemaToProto takes a *configschema.Block and converts it to a
// proto.Schema_Block for a grpc response.
func ConfigSchemaToProto(b *configschema.Block) *proto.Schema_Block {
	block := &proto.Schema_Block{}

	for _, name := range sortedKeys(b.Attributes) {
		a := b.Attributes[name]
		attr := &proto.Schema_Attribute{
			Name:        name,
			Description: a.Description,
			Optional:    a.Optional,
			Computed:    a.Computed,
			Required:    a.Required,
			Sensitive:   a.Sensitive,
		}

		ty, err := json.Marshal(a.Type)
		if err != nil {
			panic(err)
		}

		attr.Type = ty

		block.Attributes = append(block.Attributes, attr)
	}

	for _, name := range sortedKeys(b.BlockTypes) {
		b := b.BlockTypes[name]
		block.BlockTypes = append(block.BlockTypes, protoSchemaNestedBlock(name, b))
	}

	return block
}

func protoSchemaNestedBlock(name string, b *configschema.NestedBlock) *proto.Schema_NestedBlock {
	return &proto.Schema_NestedBlock{
		TypeName: name,
		Block:    ConfigSchemaToProto(&b.Block),
		Nesting:  proto.Schema_NestedBlock_NestingMode(b.Nesting),
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
	}

	for _, a := range b.Attributes {
		attr := &configschema.Attribute{
			Description: a.Description,
			Required:    a.Required,
			Optional:    a.Optional,
			Computed:    a.Computed,
			Sensitive:   a.Sensitive,
		}

		if err := json.Unmarshal(a.Type, &attr.Type); err != nil {
			panic(err)
		}

		block.Attributes[a.Name] = attr
	}

	for _, b := range b.BlockTypes {
		block.BlockTypes[b.TypeName] = schemaNestedBlock(b)
	}

	return block
}

func schemaNestedBlock(b *proto.Schema_NestedBlock) *configschema.NestedBlock {
	nb := &configschema.NestedBlock{
		Nesting:  configschema.NestingMode(b.Nesting),
		MinItems: int(b.MinItems),
		MaxItems: int(b.MaxItems),
	}

	nested := ProtoToConfigSchema(b.Block)
	nb.Block = *nested
	return nb
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
