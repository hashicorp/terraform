package plugin

import (
	"encoding/json"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plugin/proto"
)

// protoSchemaBlock takes a *configschema.Block and converts it to a
// proto.Schema_Block for a grpc response.
func protoSchemaBlock(b *configschema.Block) *proto.Schema_Block {
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
		Block:    protoSchemaBlock(&b.Block),
		Nesting:  proto.Schema_NestedBlock_NestingMode(b.Nesting),
		MinItems: int64(b.MinItems),
		MaxItems: int64(b.MaxItems),
	}
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
