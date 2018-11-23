package hclwrite

import (
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type Block struct {
	inTree

	leadComments *node
	typeName     *node
	labels       nodeSet
	open         *node
	body         *node
	close        *node
}

func newBlock() *Block {
	return &Block{
		inTree: newInTree(),
		labels: newNodeSet(),
	}
}

// NewBlock constructs a new, empty block with the given type name and labels.
func NewBlock(typeName string, labels []string) *Block {
	block := newBlock()
	block.init(typeName, labels)
	return block
}

func (b *Block) init(typeName string, labels []string) {
	nameTok := newIdentToken(typeName)
	nameObj := newIdentifier(nameTok)
	b.leadComments = b.children.Append(newComments(nil))
	b.typeName = b.children.Append(nameObj)
	for _, label := range labels {
		labelToks := TokensForValue(cty.StringVal(label))
		labelObj := newQuoted(labelToks)
		labelNode := b.children.Append(labelObj)
		b.labels.Add(labelNode)
	}
	b.open = b.children.AppendUnstructuredTokens(Tokens{
		{
			Type:  hclsyntax.TokenOBrace,
			Bytes: []byte{'{'},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	})
	body := newBody() // initially totally empty; caller can append to it subsequently
	b.body = b.children.Append(body)
	b.close = b.children.AppendUnstructuredTokens(Tokens{
		{
			Type:  hclsyntax.TokenCBrace,
			Bytes: []byte{'}'},
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	})
}

// Body returns the body that represents the content of the receiving block.
//
// Appending to or otherwise modifying this body will make changes to the
// tokens that are generated between the blocks open and close braces.
func (b *Block) Body() *Body {
	return b.body.content.(*Body)
}
