package configschema

import (
	"github.com/zclconf/go-cty/cty"
)

// AttributeByPath looks up the Attribute schema which corresponds to the given
// cty.Path. A nil value is returned if the given path does not correspond to a
// specific attribute.
// TODO: this will need to be updated for nested attributes
func (b *Block) AttributeByPath(path cty.Path) *Attribute {
	block := b
	for _, step := range path {
		switch step := step.(type) {
		case cty.GetAttrStep:
			if attr := block.Attributes[step.Name]; attr != nil {
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
