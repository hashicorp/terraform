package configschema

type FilterT[T any] func(string, T) bool

var (
	FilterReadOnlyAttribute = func(name string, attribute *Attribute) bool {
		return attribute.Computed && !attribute.Optional
	}

	FilterHelperSchemaIdAttribute = func(name string, attribute *Attribute) bool {
		if name == "id" && attribute.Computed && attribute.Optional {
			return true
		}
		return false
	}

	FilterDeprecatedAttribute = func(name string, attribute *Attribute) bool {
		return attribute.Deprecated
	}

	FilterDeprecatedBlock = func(name string, block *NestedBlock) bool {
		return block.Deprecated
	}
)

func FilterOr[T any](filters ...FilterT[T]) FilterT[T] {
	return func(name string, value T) bool {
		for _, f := range filters {
			if f(name, value) {
				return true
			}
		}
		return false
	}
}

func (b *Block) Filter(filterAttribute FilterT[*Attribute], filterBlock FilterT[*NestedBlock]) *Block {
	ret := &Block{
		Description:     b.Description,
		DescriptionKind: b.DescriptionKind,
		Deprecated:      b.Deprecated,
	}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, attrS := range b.Attributes {
		if filterAttribute == nil || !filterAttribute(name, attrS) {
			ret.Attributes[name] = attrS
		}

		if attrS.NestedType != nil {
			ret.Attributes[name].NestedType = filterNestedType(attrS.NestedType, filterAttribute)
		}
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		if filterBlock == nil || !filterBlock(name, blockS) {
			block := blockS.Filter(filterAttribute, filterBlock)
			ret.BlockTypes[name] = &NestedBlock{
				Block:    *block,
				Nesting:  blockS.Nesting,
				MinItems: blockS.MinItems,
				MaxItems: blockS.MaxItems,
			}
		}
	}

	return ret
}

func filterNestedType(obj *Object, filterAttribute FilterT[*Attribute]) *Object {
	if obj == nil {
		return nil
	}

	ret := &Object{
		Attributes: map[string]*Attribute{},
		Nesting:    obj.Nesting,
	}

	for name, attrS := range obj.Attributes {
		if filterAttribute == nil || !filterAttribute(name, attrS) {
			ret.Attributes[name] = attrS
			if attrS.NestedType != nil {
				ret.Attributes[name].NestedType = filterNestedType(attrS.NestedType, filterAttribute)
			}
		}
	}

	return ret
}
