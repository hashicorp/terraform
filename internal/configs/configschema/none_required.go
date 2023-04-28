package configschema

// NoneRequired returns a deep copy of the receiver with any required
// attributes translated to optional.
func (b *Block) NoneRequired() *Block {
	ret := &Block{}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, attrS := range b.Attributes {
		ret.Attributes[name] = attrS.forceOptional()
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		ret.BlockTypes[name] = blockS.noneRequired()
	}

	return ret
}

func (b *NestedBlock) noneRequired() *NestedBlock {
	ret := *b
	ret.Block = *(ret.Block.NoneRequired())
	ret.MinItems = 0
	ret.MaxItems = 0
	return &ret
}

func (a *Attribute) forceOptional() *Attribute {
	ret := *a
	ret.Optional = true
	ret.Required = false
	return &ret
}

// OmitAttr returns a deep copy of the receiver removing any top-level attribute
// with the supplied name.
func (b *Block) OmitAttr(name string) *Block {
	ret := &Block{}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for attrN, attrS := range b.Attributes {
		if attrN != name {
			ret.Attributes[attrN] = attrS
		}
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		ret.BlockTypes[name] = blockS
	}

	return ret
}

// OmitReadOnly returns a deep copy of the receiver filtering out any attributes
// that are both computed and non-optional.
func (b *Block) OmitReadOnly() *Block {
	ret := &Block{}

	if b.Attributes != nil {
		ret.Attributes = make(map[string]*Attribute, len(b.Attributes))
	}
	for name, attrS := range b.Attributes {
		if !(attrS.Computed && !attrS.Optional) {
			ret.Attributes[name] = attrS
		}
	}

	if b.BlockTypes != nil {
		ret.BlockTypes = make(map[string]*NestedBlock, len(b.BlockTypes))
	}
	for name, blockS := range b.BlockTypes {
		ret.BlockTypes[name] = blockS.omitReadOnly()
	}

	return ret
}

func (b *NestedBlock) omitReadOnly() *NestedBlock {
	ret := *b
	ret.Block = *(ret.Block.OmitReadOnly())
	return &ret
}
