// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configschema

// DeepCopy returns a deep copy of the schema.
func (b *Block) DeepCopy() *Block {
	block := &Block{
		Description:     b.Description,
		DescriptionKind: b.DescriptionKind,
		Deprecated:      b.Deprecated,
	}

	if b.Attributes != nil {
		block.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, a := range b.Attributes {
		block.Attributes[name] = a.DeepCopy()
	}

	if b.BlockTypes != nil {
		block.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, bt := range b.BlockTypes {
		inner := bt.Block.DeepCopy()
		block.BlockTypes[name] = &NestedBlock{
			Block:    *inner,
			Nesting:  bt.Nesting,
			MinItems: bt.MinItems,
			MaxItems: bt.MaxItems,
		}
	}

	return block
}

// DeepCopy returns a deep copy of the schema.
func (a *Attribute) DeepCopy() *Attribute {
	attr := &Attribute{
		Type:            a.Type,
		Description:     a.Description,
		DescriptionKind: a.DescriptionKind,
		Deprecated:      a.Deprecated,
		Required:        a.Required,
		Computed:        a.Computed,
		Optional:        a.Optional,
		Sensitive:       a.Sensitive,

		// NestedType is not copied here because it will be copied
		// separately if it is set.
		NestedType: nil,
	}
	if a.NestedType != nil {
		attr.NestedType = a.NestedType.DeepCopy()
	}
	return attr
}

// DeepCopy returns a deep copy of the schema.
func (o *Object) DeepCopy() *Object {
	object := &Object{
		Nesting: o.Nesting,
	}
	if o.Attributes != nil {
		object.Attributes = make(map[string]*Attribute, len(o.Attributes))
		for name, a := range o.Attributes {
			object.Attributes[name] = a.DeepCopy()
		}
	}
	return object
}
