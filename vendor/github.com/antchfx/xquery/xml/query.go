/*
Package xmlquery provides extract data from XML documents using XPath expression.
*/
package xmlquery

import (
	"fmt"
	"strings"

	"github.com/antchfx/xpath"
)

// SelectElements finds child elements with the specified name.
func (n *Node) SelectElements(name string) []*Node {
	return Find(n, name)
}

// SelectElement finds child elements with the specified name.
func (n *Node) SelectElement(name string) *Node {
	return FindOne(n, name)
}

// SelectAttr returns the attribute value with the specified name.
func (n *Node) SelectAttr(name string) string {
	var local, space string
	local = name
	if i := strings.Index(name, ":"); i > 0 {
		space = name[:i]
		local = name[i+1:]
	}
	for _, attr := range n.Attr {
		if attr.Name.Local == local && attr.Name.Space == space {
			return attr.Value
		}
	}
	return ""
}

var _ xpath.NodeNavigator = &NodeNavigator{}

// CreateXPathNavigator creates a new xpath.NodeNavigator for the specified html.Node.
func CreateXPathNavigator(top *Node) *NodeNavigator {
	return &NodeNavigator{curr: top, root: top, attr: -1}
}

// Find searches the Node that matches by the specified XPath expr.
func Find(top *Node, expr string) []*Node {
	exp, err := xpath.Compile(expr)
	if err != nil {
		panic(err)
	}
	t := exp.Select(CreateXPathNavigator(top))
	var elems []*Node
	for t.MoveNext() {
		elems = append(elems, (t.Current().(*NodeNavigator)).curr)
	}
	return elems
}

// FindOne searches the Node that matches by the specified XPath expr,
// and returns first element of matched.
func FindOne(top *Node, expr string) *Node {
	exp, err := xpath.Compile(expr)
	if err != nil {
		panic(err)
	}
	t := exp.Select(CreateXPathNavigator(top))
	var elem *Node
	if t.MoveNext() {
		elem = (t.Current().(*NodeNavigator)).curr
	}
	return elem
}

// FindEach searches the html.Node and calls functions cb.
func FindEach(top *Node, expr string, cb func(int, *Node)) {
	exp, err := xpath.Compile(expr)
	if err != nil {
		panic(err)
	}
	t := exp.Select(CreateXPathNavigator(top))
	var i int
	for t.MoveNext() {
		cb(i, (t.Current().(*NodeNavigator)).curr)
		i++
	}
}

type NodeNavigator struct {
	root, curr *Node
	attr       int
}

func (x *NodeNavigator) Current() *Node {
	return x.curr
}

func (x *NodeNavigator) NodeType() xpath.NodeType {
	switch x.curr.Type {
	case CommentNode:
		return xpath.CommentNode
	case TextNode:
		return xpath.TextNode
	case DeclarationNode, DocumentNode:
		return xpath.RootNode
	case ElementNode:
		if x.attr != -1 {
			return xpath.AttributeNode
		}
		return xpath.ElementNode
	}
	panic(fmt.Sprintf("unknown XML node type: %v", x.curr.Type))
}

func (x *NodeNavigator) LocalName() string {
	if x.attr != -1 {
		return x.curr.Attr[x.attr].Name.Local
	}
	return x.curr.Data

}

func (x *NodeNavigator) Prefix() string {
	return x.curr.Prefix
}

func (x *NodeNavigator) Value() string {
	switch x.curr.Type {
	case CommentNode:
		return x.curr.Data
	case ElementNode:
		if x.attr != -1 {
			return x.curr.Attr[x.attr].Value
		}
		return x.curr.InnerText()
	case TextNode:
		return x.curr.Data
	}
	return ""
}

func (x *NodeNavigator) Copy() xpath.NodeNavigator {
	n := *x
	return &n
}

func (x *NodeNavigator) MoveToRoot() {
	x.curr = x.root
}

func (x *NodeNavigator) MoveToParent() bool {
	if x.attr != -1 {
		x.attr = -1
		return true
	} else if node := x.curr.Parent; node != nil {
		x.curr = node
		return true
	}
	return false
}

func (x *NodeNavigator) MoveToNextAttribute() bool {
	if x.attr >= len(x.curr.Attr)-1 {
		return false
	}
	x.attr++
	return true
}

func (x *NodeNavigator) MoveToChild() bool {
	if x.attr != -1 {
		return false
	}
	if node := x.curr.FirstChild; node != nil {
		x.curr = node
		return true
	}
	return false
}

func (x *NodeNavigator) MoveToFirst() bool {
	if x.attr != -1 || x.curr.PrevSibling == nil {
		return false
	}
	for {
		node := x.curr.PrevSibling
		if node == nil {
			break
		}
		x.curr = node
	}
	return true
}

func (x *NodeNavigator) String() string {
	return x.Value()
}

func (x *NodeNavigator) MoveToNext() bool {
	if x.attr != -1 {
		return false
	}
	if node := x.curr.NextSibling; node != nil {
		x.curr = node
		return true
	}
	return false
}

func (x *NodeNavigator) MoveToPrevious() bool {
	if x.attr != -1 {
		return false
	}
	if node := x.curr.PrevSibling; node != nil {
		x.curr = node
		return true
	}
	return false
}

func (x *NodeNavigator) MoveTo(other xpath.NodeNavigator) bool {
	node, ok := other.(*NodeNavigator)
	if !ok || node.root != x.root {
		return false
	}

	x.curr = node.curr
	x.attr = node.attr
	return true
}
