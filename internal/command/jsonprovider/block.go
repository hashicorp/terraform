// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonprovider

import (
	"github.com/hashicorp/terraform/internal/configs/configschema"
)

type Block struct {
	Attributes      map[string]*Attribute `json:"attributes,omitempty"`
	BlockTypes      map[string]*BlockType `json:"block_types,omitempty"`
	Description     string                `json:"description,omitempty"`
	DescriptionKind string                `json:"description_kind,omitempty"`
	Deprecated      bool                  `json:"deprecated,omitempty"`
}

type BlockType struct {
	NestingMode string `json:"nesting_mode,omitempty"`
	Block       *Block `json:"block,omitempty"`
	MinItems    uint64 `json:"min_items,omitempty"`
	MaxItems    uint64 `json:"max_items,omitempty"`
}

func marshalBlockTypes(nestedBlock *configschema.NestedBlock) *BlockType {
	if nestedBlock == nil {
		return &BlockType{}
	}
	ret := &BlockType{
		Block:       marshalBlock(&nestedBlock.Block),
		MinItems:    uint64(nestedBlock.MinItems),
		MaxItems:    uint64(nestedBlock.MaxItems),
		NestingMode: nestingModeString(nestedBlock.Nesting),
	}
	return ret
}

func marshalBlock(configBlock *configschema.Block) *Block {
	if configBlock == nil {
		return &Block{}
	}

	ret := Block{
		Deprecated:      configBlock.Deprecated,
		Description:     configBlock.Description,
		DescriptionKind: marshalStringKind(configBlock.DescriptionKind),
	}

	if len(configBlock.Attributes) > 0 {
		attrs := make(map[string]*Attribute, len(configBlock.Attributes))
		for k, attr := range configBlock.Attributes {
			attrs[k] = marshalAttribute(attr)
		}
		ret.Attributes = attrs
	}

	if len(configBlock.BlockTypes) > 0 {
		blockTypes := make(map[string]*BlockType, len(configBlock.BlockTypes))
		for k, bt := range configBlock.BlockTypes {
			blockTypes[k] = marshalBlockTypes(bt)
		}
		ret.BlockTypes = blockTypes
	}

	return &ret
}

func nestingModeString(mode configschema.NestingMode) string {
	switch mode {
	case configschema.NestingSingle:
		return "single"
	case configschema.NestingGroup:
		return "group"
	case configschema.NestingList:
		return "list"
	case configschema.NestingSet:
		return "set"
	case configschema.NestingMap:
		return "map"
	default:
		return "invalid"
	}
}
