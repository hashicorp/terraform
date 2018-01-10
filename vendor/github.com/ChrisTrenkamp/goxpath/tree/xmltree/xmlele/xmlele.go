package xmlele

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/tree"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree/xmlbuilder"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree/xmlnode"
)

//XMLEle is an implementation of XPRes for XML elements
type XMLEle struct {
	xml.StartElement
	tree.NSBuilder
	Attrs    []tree.Node
	Children []tree.Node
	Parent   tree.Elem
	tree.NodePos
	tree.NodeType
}

//Root is the default root node builder for xmltree.ParseXML
func Root() xmlbuilder.XMLBuilder {
	return &XMLEle{NodeType: tree.NtRoot}
}

//CreateNode is an implementation of xmlbuilder.XMLBuilder.  It appends the node
//specified in opts and returns the child if it is an element.  Otherwise, it returns x.
func (x *XMLEle) CreateNode(opts *xmlbuilder.BuilderOpts) xmlbuilder.XMLBuilder {
	if opts.NodeType == tree.NtElem {
		ele := &XMLEle{
			StartElement: opts.Tok.(xml.StartElement),
			NSBuilder:    tree.NSBuilder{NS: opts.NS},
			Attrs:        make([]tree.Node, len(opts.Attrs)),
			Parent:       x,
			NodePos:      tree.NodePos(opts.NodePos),
			NodeType:     opts.NodeType,
		}
		for i := range opts.Attrs {
			ele.Attrs[i] = xmlnode.XMLNode{
				Token:    opts.Attrs[i],
				NodePos:  tree.NodePos(opts.AttrStartPos + i),
				NodeType: tree.NtAttr,
				Parent:   ele,
			}
		}
		x.Children = append(x.Children, ele)
		return ele
	}

	node := xmlnode.XMLNode{
		Token:    opts.Tok,
		NodePos:  tree.NodePos(opts.NodePos),
		NodeType: opts.NodeType,
		Parent:   x,
	}
	x.Children = append(x.Children, node)
	return x
}

//EndElem is an implementation of xmlbuilder.XMLBuilder.  It returns x's parent.
func (x *XMLEle) EndElem() xmlbuilder.XMLBuilder {
	return x.Parent.(*XMLEle)
}

//GetToken returns the xml.Token representation of the node
func (x *XMLEle) GetToken() xml.Token {
	return x.StartElement
}

//GetParent returns the parent node, or itself if it's the root
func (x *XMLEle) GetParent() tree.Elem {
	return x.Parent
}

//GetChildren returns all child nodes of the element
func (x *XMLEle) GetChildren() []tree.Node {
	ret := make([]tree.Node, len(x.Children))

	for i := range x.Children {
		ret[i] = x.Children[i]
	}

	return ret
}

//GetAttrs returns all attributes of the element
func (x *XMLEle) GetAttrs() []tree.Node {
	ret := make([]tree.Node, len(x.Attrs))
	for i := range x.Attrs {
		ret[i] = x.Attrs[i]
	}
	return ret
}

//ResValue returns the string value of the element and children
func (x *XMLEle) ResValue() string {
	ret := ""
	for i := range x.Children {
		switch x.Children[i].GetNodeType() {
		case tree.NtChd, tree.NtElem, tree.NtRoot:
			ret += x.Children[i].ResValue()
		}
	}
	return ret
}
