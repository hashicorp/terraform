// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"math/big"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// AttributeByPath looks up the Attribute schema which corresponds to the given
// cty.Path. A nil value is returned if the given path does not correspond to a
// specific attribute.
func (b *Block) AttributeByPath(path cty.Path) *Attribute {
	block := b
	for i, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if attr := block.Attributes[step.Name]; attr != nil {
				// If the Attribute is defined with a NestedType and there's
				// more to the path, descend into the NestedType
				if attr.NestedType != nil && i < len(path)-1 {
					return attr.NestedType.AttributeByPath(path[i+1:])
				} else if i < len(path)-1 { // There's more to the path, but not more to this Attribute.
					return nil
				}
				return attr
			}

			if nestedBlock := block.BlockTypes[step.Name]; nestedBlock != nil {
				block = &nestedBlock.Block
				continue
			}

			return nil
		}
	}
	return nil
}

// AttributeByPath recurses through a NestedType to look up the Attribute scheme
// which corresponds to the given cty.Path. A nil value is returned if the given
// path does not correspond to a specific attribute.
func (o *Object) AttributeByPath(path cty.Path) *Attribute {
	for i, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if attr := o.Attributes[step.Name]; attr != nil {
				if attr.NestedType != nil && i < len(path)-1 {
					return attr.NestedType.AttributeByPath(path[i+1:])
				} else if i < len(path)-1 { // There's more to the path, but not more to this Attribute.
					return nil
				}
				return attr
			}
		}
	}
	return nil
}

// NestedBlockByPath looks up the Block schema which corresponds to the given
// cty.Path. A nil value is returned if the given path does not correspond to a
// specific attribute.
func (b *Block) NestedBlockByPath(path cty.Path) *NestedBlock {
	for i, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if blockType := b.BlockTypes[step.Name]; blockType != nil {
				if len(blockType.Block.BlockTypes) > 0 && i < len(path)-1 {
					return blockType.Block.NestedBlockByPath(path[i+1:])
				} else if i < len(path)-1 {
					return nil
				}
				return blockType
			}
		}
	}
	return nil
}

// WalkPath follows a static path through the declared schema, visiting each named
// attribute or nested block in order. Exactly one of the callback's attribute
// and block arguments is non-nil.
// The result is the selected type. Invalid paths and callback errors stop the
// walk and return cty.NilType.
func (b *Block) WalkPath(path cty.Path, visit func(cty.Path, *Attribute, *NestedBlock) error) (cty.Type, error) {
	if len(path) == 0 {
		return cty.NilType, path.NewErrorf("path must not be empty")
	}
	for i, step := range path {
		if index, ok := step.(cty.IndexStep); ok {
			if !index.Key.IsKnown() || index.Key.IsNull() {
				return cty.NilType, path[:i+1].NewErrorf("index must be known and non-null")
			}
			if index.Key.Type() == cty.Number {
				key, _ := index.Key.Unmark()
				n, accuracy := key.AsBigFloat().Int64()
				if accuracy != big.Exact || n < 0 {
					return cty.NilType, path[:i+1].NewErrorf("index must be a non-negative integer within int64 range")
				}
			}
		}
	}
	schema := b
	for i := 0; i < len(path); {
		if schema == nil {
			return cty.NilType, path[:i].NewErrorf("block schema is nil")
		}
		var name string
		switch step := path[i].(type) {
		case cty.GetAttrStep:
			name = step.Name
		case cty.IndexStep:
			if step.Key.Type() != cty.String || !step.Key.IsKnown() || step.Key.IsNull() {
				return cty.NilType, path[:i+1].NewErrorf("expected an attribute or block name")
			}
			key, _ := step.Key.Unmark()
			name = key.AsString()
		default:
			return cty.NilType, path[:i+1].NewErrorf("expected an attribute or block name")
		}
		i++
		attr := schema.Attributes[name]
		var block *NestedBlock
		if attr == nil {
			block = schema.BlockTypes[name]
		}
		if attr == nil && block == nil {
			return cty.NilType, path[:i].NewErrorf("no attribute or block named %q", name)
		}
		if visit != nil {
			if err := visit(path[:i].Copy(), attr, block); err != nil {
				return cty.NilType, err
			}
		}
		var nesting NestingMode
		if attr != nil {
			if i == len(path) {
				return attr.ImpliedType(), nil
			}
			if attr.NestedType == nil {
				if typ, ok := TypeAtPath(attr.ImpliedType(), path[i:]); ok {
					return typ, nil
				}
				return cty.NilType, path.NewErrorf("invalid path within attribute %q", name)
			}
			nesting = attr.NestedType.Nesting
			schema = &Block{Attributes: attr.NestedType.Attributes}
		} else {
			nesting = block.Nesting
			schema = &block.Block
			if i == len(path) {
				typ := schema.schemaType()
				switch nesting {
				case NestingList:
					typ = cty.List(typ)
				case NestingMap:
					typ = cty.Map(typ)
				case NestingSet:
					typ = cty.Set(typ)
				}
				return typ, nil
			}
		}
		switch nesting {
		case NestingSingle, NestingGroup:
			continue
		case NestingList, NestingMap:
			collectionType := cty.List(cty.DynamicPseudoType)
			if nesting == NestingMap {
				collectionType = cty.Map(cty.DynamicPseudoType)
			}
			if _, ok := TypeAtPath(collectionType, path[i:i+1]); !ok {
				return cty.NilType, path[:i+1].NewErrorf("invalid collection element")
			}
			i++
		default:
			return cty.NilType, path[:i].NewErrorf("cannot descend into this nesting mode")
		}
	}
	return schema.schemaType(), nil
}

// TypeAtPath resolves a static path through the block's declared schema,
// preserving nested block structure even when it contains dynamic attributes.
// The result can be any type, including a composite or dynamic type. Empty or
// invalid paths, set elements, and unknown or null indices are rejected.
func (b *Block) TypeAtPath(path cty.Path) (cty.Type, bool) {
	typ, err := b.WalkPath(path, nil)
	return typ, err == nil
}

// TypeAtPath resolves a static path through a schema's type. Object and map
// members accept both attribute steps and string index steps.
func TypeAtPath(typ cty.Type, path cty.Path) (cty.Type, bool) {
	if len(path) == 0 {
		return cty.NilType, false
	}
	value := cty.UnknownVal(typ)
	for _, step := range path {
		// A wholly dynamic value cannot establish the type of a descendant.
		if value.Type() == cty.DynamicPseudoType {
			return cty.NilType, false
		}

		switch step := step.(type) {
		case cty.GetAttrStep:
			// HCL validates named access for the current value type.
		case cty.IndexStep:
			// HCL accepts unknown keys and coerces index types. A static schema
			// path must instead have a known key of the collection's key type.
			if !step.Key.IsKnown() || step.Key.IsNull() {
				return cty.NilType, false
			}
			if step.Key.Type() == cty.Number {
				number, _ := step.Key.Unmark()
				n, accuracy := number.AsBigFloat().Int64()
				if accuracy != big.Exact || n < 0 {
					return cty.NilType, false
				}
			}
			typ := value.Type()
			if (typ.IsListType() || typ.IsTupleType()) && step.Key.Type() != cty.Number {
				return cty.NilType, false
			}
			if (typ.IsMapType() || typ.IsObjectType()) && step.Key.Type() != cty.String {
				return cty.NilType, false
			}
		default:
			return cty.NilType, false
		}

		// Like StaticValidateTraversal, use HCL to apply value-type suffixes
		// to an unknown value rather than reproducing its type traversal rules.
		var diags hcl.Diagnostics
		value, diags = hcl.ApplyPath(value, cty.Path{step}, nil)
		if diags.HasErrors() {
			return cty.NilType, false
		}
	}
	return value.Type(), true
}
