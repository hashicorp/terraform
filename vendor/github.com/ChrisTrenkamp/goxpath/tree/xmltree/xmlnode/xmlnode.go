package xmlnode

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

//XMLNode will represent an attribute, character data, comment, or processing instruction node
type XMLNode struct {
	xml.Token
	tree.NodePos
	tree.NodeType
	Parent tree.Elem
}

//GetToken returns the xml.Token representation of the node
func (a XMLNode) GetToken() xml.Token {
	if a.NodeType == tree.NtAttr {
		ret := a.Token.(*xml.Attr)
		return *ret
	}
	return a.Token
}

//GetParent returns the parent node
func (a XMLNode) GetParent() tree.Elem {
	return a.Parent
}

//ResValue returns the string value of the attribute
func (a XMLNode) ResValue() string {
	switch a.NodeType {
	case tree.NtAttr:
		return a.Token.(*xml.Attr).Value
	case tree.NtChd:
		return string(a.Token.(xml.CharData))
	case tree.NtComm:
		return string(a.Token.(xml.Comment))
	}
	//case tree.NtPi:
	return string(a.Token.(xml.ProcInst).Inst)
}
