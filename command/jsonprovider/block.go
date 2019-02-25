package jsonprovider

import (
	"github.com/hashicorp/terraform/configs/configschema"
)

type block struct {
	Attributes map[string]*attribute `json:"attributes,omitempty"`
	BlockTypes map[string]*blockType `json:"block_types,omitempty"`
}

type blockType struct {
	NestingMode string `json:"nesting_mode,omitempty"`
	Block       *block `json:"block,omitempty"`
	MinItems    uint64 `json:"min_items,omitempty"`
	MaxItems    uint64 `json:"max_items,omitempty"`
}

func marshalBlockTypes(nestedBlock *configschema.NestedBlock) *blockType {
	if nestedBlock == nil {
		return &blockType{}
	}
	ret := &blockType{
		Block:    marshalBlock(&nestedBlock.Block),
		MinItems: uint64(nestedBlock.MinItems),
		MaxItems: uint64(nestedBlock.MaxItems),
	}

	switch nestedBlock.Nesting {
	case configschema.NestingSingle:
		ret.NestingMode = "single"
	case configschema.NestingList:
		ret.NestingMode = "list"
	case configschema.NestingSet:
		ret.NestingMode = "set"
	case configschema.NestingMap:
		ret.NestingMode = "map"
	default:
		ret.NestingMode = "invalid"
	}
	return ret
}

func marshalBlock(configBlock *configschema.Block) *block {
	if configBlock == nil {
		return &block{}
	}

	var ret block
	if len(configBlock.Attributes) > 0 {
		attrs := make(map[string]*attribute, len(configBlock.Attributes))
		for k, attr := range configBlock.Attributes {
			attrs[k] = marshalAttribute(attr)
		}
		ret.Attributes = attrs
	}

	if len(configBlock.BlockTypes) > 0 {
		blockTypes := make(map[string]*blockType, len(configBlock.BlockTypes))
		for k, bt := range configBlock.BlockTypes {
			blockTypes[k] = marshalBlockTypes(bt)
		}
		ret.BlockTypes = blockTypes
	}

	return &ret
}
