package tree

import (
	"encoding/xml"
	"sort"
)

//XMLSpace is the W3C XML namespace
const XMLSpace = "http://www.w3.org/XML/1998/namespace"

//NodePos is a helper for representing the node's document order
type NodePos int

//Pos returns the node's document order position
func (n NodePos) Pos() int {
	return int(n)
}

//NodeType is a safer way to determine a node's type than type assertions.
type NodeType int

//GetNodeType returns the node's type.
func (t NodeType) GetNodeType() NodeType {
	return t
}

//These are all the possible node types
const (
	NtAttr NodeType = iota
	NtChd
	NtComm
	NtElem
	NtNs
	NtRoot
	NtPi
)

//Node is a XPath result that is a node except elements
type Node interface {
	//ResValue prints the node's string value
	ResValue() string
	//Pos returns the node's position in the document order
	Pos() int
	//GetToken returns the xml.Token representation of the node
	GetToken() xml.Token
	//GetParent returns the parent node, which will always be an XML element
	GetParent() Elem
	//GetNodeType returns the node's type
	GetNodeType() NodeType
}

//Elem is a XPath result that is an element node
type Elem interface {
	Node
	//GetChildren returns the elements children.
	GetChildren() []Node
	//GetAttrs returns the attributes of the element
	GetAttrs() []Node
}

//NSElem is a node that keeps track of namespaces.
type NSElem interface {
	Elem
	GetNS() map[xml.Name]string
}

//NSBuilder is a helper-struct for satisfying the NSElem interface
type NSBuilder struct {
	NS map[xml.Name]string
}

//GetNS returns the namespaces found on the current element.  It should not be
//confused with BuildNS, which actually resolves the namespace nodes.
func (ns NSBuilder) GetNS() map[xml.Name]string {
	return ns.NS
}

type nsValueSort []NS

func (ns nsValueSort) Len() int { return len(ns) }
func (ns nsValueSort) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}
func (ns nsValueSort) Less(i, j int) bool {
	return ns[i].Value < ns[j].Value
}

//BuildNS resolves all the namespace nodes of the element and returns them
func BuildNS(t Elem) (ret []NS) {
	vals := make(map[xml.Name]string)

	if nselem, ok := t.(NSElem); ok {
		buildNS(nselem, vals)

		ret = make([]NS, 0, len(vals))
		i := 1

		for k, v := range vals {
			if !(k.Local == "xmlns" && k.Space == "" && v == "") {
				ret = append(ret, NS{
					Attr:     xml.Attr{Name: k, Value: v},
					Parent:   t,
					NodeType: NtNs,
				})
				i++
			}
		}

		sort.Sort(nsValueSort(ret))
		for i := range ret {
			ret[i].NodePos = NodePos(t.Pos() + i + 1)
		}
	}

	return ret
}

func buildNS(x NSElem, ret map[xml.Name]string) {
	if x.GetNodeType() == NtRoot {
		return
	}

	if nselem, ok := x.GetParent().(NSElem); ok {
		buildNS(nselem, ret)
	}

	for k, v := range x.GetNS() {
		ret[k] = v
	}
}

//NS is a namespace node.
type NS struct {
	xml.Attr
	Parent Elem
	NodePos
	NodeType
}

//GetToken returns the xml.Token representation of the namespace.
func (ns NS) GetToken() xml.Token {
	return ns.Attr
}

//GetParent returns the parent node of the namespace.
func (ns NS) GetParent() Elem {
	return ns.Parent
}

//ResValue returns the string value of the namespace
func (ns NS) ResValue() string {
	return ns.Attr.Value
}

//GetAttribute is a convenience function for getting the specified attribute from an element.
//false is returned if the attribute is not found.
func GetAttribute(n Elem, local, space string) (xml.Attr, bool) {
	attrs := n.GetAttrs()
	for _, i := range attrs {
		attr := i.GetToken().(xml.Attr)
		if local == attr.Name.Local && space == attr.Name.Space {
			return attr, true
		}
	}
	return xml.Attr{}, false
}

//GetAttributeVal is like GetAttribute, except it returns the attribute's value.
func GetAttributeVal(n Elem, local, space string) (string, bool) {
	attr, ok := GetAttribute(n, local, space)
	return attr.Value, ok
}

//GetAttrValOrEmpty is like GetAttributeVal, except it returns an empty string if
//the attribute is not found instead of false.
func GetAttrValOrEmpty(n Elem, local, space string) string {
	val, ok := GetAttributeVal(n, local, space)
	if !ok {
		return ""
	}
	return val
}

//FindNodeByPos finds a node from the given position.  Returns nil if the node
//is not found.
func FindNodeByPos(n Node, pos int) Node {
	if n.Pos() == pos {
		return n
	}

	if elem, ok := n.(Elem); ok {
		chldrn := elem.GetChildren()
		for i := 1; i < len(chldrn); i++ {
			if chldrn[i-1].Pos() <= pos && chldrn[i].Pos() > pos {
				return FindNodeByPos(chldrn[i-1], pos)
			}
		}

		if len(chldrn) > 0 {
			if chldrn[len(chldrn)-1].Pos() <= pos {
				return FindNodeByPos(chldrn[len(chldrn)-1], pos)
			}
		}

		attrs := elem.GetAttrs()
		for _, i := range attrs {
			if i.Pos() == pos {
				return i
			}
		}

		ns := BuildNS(elem)
		for _, i := range ns {
			if i.Pos() == pos {
				return i
			}
		}
	}

	return nil
}
