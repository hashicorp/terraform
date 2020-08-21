package configschema

// NoneComputed returns a deep copy of the receiver with any computed
// attributes removed.
func (b *Block) NoneComputed() *Block {
	ret := &Block{}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, attrS := range b.Attributes {
		if attrS.Computed {
			continue
		}
		attr := *attrS
		ret.Attributes[name] = &attr
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		ret.BlockTypes[name] = blockS.noneComputed()
	}

	return ret
}

func (b *NestedBlock) noneComputed() *NestedBlock {
	ret := *b
	ret.Block = *(ret.Block.NoneComputed())
	ret.MinItems = 0
	ret.MaxItems = 0
	return &ret
}
