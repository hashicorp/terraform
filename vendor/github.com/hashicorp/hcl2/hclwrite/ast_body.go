package hclwrite

import (
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

type Body struct {
	inTree

	items nodeSet
}

func newBody() *Body {
	return &Body{
		inTree: newInTree(),
		items:  newNodeSet(),
	}
}

func (b *Body) appendItem(c nodeContent) *node {
	nn := b.children.Append(c)
	b.items.Add(nn)
	return nn
}

func (b *Body) appendItemNode(nn *node) *node {
	nn.assertUnattached()
	b.children.AppendNode(nn)
	b.items.Add(nn)
	return nn
}

// Clear removes all of the items from the body, making it empty.
func (b *Body) Clear() {
	b.children.Clear()
}

func (b *Body) AppendUnstructuredTokens(ts Tokens) {
	b.inTree.children.Append(ts)
}

// Attributes returns a new map of all of the attributes in the body, with
// the attribute names as the keys.
func (b *Body) Attributes() map[string]*Attribute {
	ret := make(map[string]*Attribute)
	for n := range b.items {
		if attr, isAttr := n.content.(*Attribute); isAttr {
			nameObj := attr.name.content.(*identifier)
			name := string(nameObj.token.Bytes)
			ret[name] = attr
		}
	}
	return ret
}

// Blocks returns a new slice of all the blocks in the body.
func (b *Body) Blocks() []*Block {
	ret := make([]*Block, 0, len(b.items))
	for n := range b.items {
		if block, isBlock := n.content.(*Block); isBlock {
			ret = append(ret, block)
		}
	}
	return ret
}

// GetAttribute returns the attribute from the body that has the given name,
// or returns nil if there is currently no matching attribute.
func (b *Body) GetAttribute(name string) *Attribute {
	for n := range b.items {
		if attr, isAttr := n.content.(*Attribute); isAttr {
			nameObj := attr.name.content.(*identifier)
			if nameObj.hasName(name) {
				// We've found it!
				return attr
			}
		}
	}

	return nil
}

// SetAttributeValue either replaces the expression of an existing attribute
// of the given name or adds a new attribute definition to the end of the block.
//
// The value is given as a cty.Value, and must therefore be a literal. To set
// a variable reference or other traversal, use SetAttributeTraversal.
//
// The return value is the attribute that was either modified in-place or
// created.
func (b *Body) SetAttributeValue(name string, val cty.Value) *Attribute {
	attr := b.GetAttribute(name)
	expr := NewExpressionLiteral(val)
	if attr != nil {
		attr.expr = attr.expr.ReplaceWith(expr)
	} else {
		attr := newAttribute()
		attr.init(name, expr)
		b.appendItem(attr)
	}
	return attr
}

// SetAttributeTraversal either replaces the expression of an existing attribute
// of the given name or adds a new attribute definition to the end of the body.
//
// The new expression is given as a hcl.Traversal, which must be an absolute
// traversal. To set a literal value, use SetAttributeValue.
//
// The return value is the attribute that was either modified in-place or
// created.
func (b *Body) SetAttributeTraversal(name string, traversal hcl.Traversal) *Attribute {
	attr := b.GetAttribute(name)
	expr := NewExpressionAbsTraversal(traversal)
	if attr != nil {
		attr.expr = attr.expr.ReplaceWith(expr)
	} else {
		attr := newAttribute()
		attr.init(name, expr)
		b.appendItem(attr)
	}
	return attr
}

// AppendBlock appends an existing block (which must not be already attached
// to a body) to the end of the receiving body.
func (b *Body) AppendBlock(block *Block) *Block {
	b.appendItem(block)
	return block
}

// AppendNewBlock appends a new nested block to the end of the receiving body
// with the given type name and labels.
func (b *Body) AppendNewBlock(typeName string, labels []string) *Block {
	block := newBlock()
	block.init(typeName, labels)
	b.appendItem(block)
	return block
}

// AppendNewline appends a newline token to th end of the receiving body,
// which generally serves as a separator between different sets of body
// contents.
func (b *Body) AppendNewline() {
	b.AppendUnstructuredTokens(Tokens{
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte{'\n'},
		},
	})
}
