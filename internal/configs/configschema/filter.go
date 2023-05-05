package configschema

type FilterT[T any] func(string, T) bool

var (
	FilterReadOnlyAttributes = func(name string, attribute *Attribute) bool {
		return attribute.Computed && !attribute.Optional
	}

	FilterDeprecatedAttribute = func(name string, attribute *Attribute) bool {
		return attribute.Deprecated
	}

	FilterDeprecatedBlock = func(name string, block *NestedBlock) bool {
		return block.Deprecated
	}
)

func FilterOr[T any](one, two FilterT[T]) FilterT[T] {
	return func(name string, value T) bool {
		return one(name, value) || two(name, value)
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
