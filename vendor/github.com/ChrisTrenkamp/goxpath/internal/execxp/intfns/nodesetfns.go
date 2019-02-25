package intfns

import (
	"encoding/xml"
	"fmt"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

func last(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	return tree.Num(c.Size), nil
}

func position(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	return tree.Num(c.Pos), nil
}

func count(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	n, ok := args[0].(tree.NodeSet)
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a node-set")
	}

	return tree.Num(len(n)), nil
}

func localName(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	var n tree.NodeSet
	ok := true
	if len(args) == 1 {
		n, ok = args[0].(tree.NodeSet)
	} else {
		n = c.NodeSet
	}
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a node-set")
	}

	ret := ""
	if len(n) == 0 {
		return tree.String(ret), nil
	}
	node := n[0]

	tok := node.GetToken()

	switch node.GetNodeType() {
	case tree.NtElem:
		ret = tok.(xml.StartElement).Name.Local
	case tree.NtAttr:
		ret = tok.(xml.Attr).Name.Local
	case tree.NtPi:
		ret = tok.(xml.ProcInst).Target
	}

	return tree.String(ret), nil
}

func namespaceURI(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	var n tree.NodeSet
	ok := true
	if len(args) == 1 {
		n, ok = args[0].(tree.NodeSet)
	} else {
		n = c.NodeSet
	}
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a node-set")
	}

	ret := ""
	if len(n) == 0 {
		return tree.String(ret), nil
	}
	node := n[0]

	tok := node.GetToken()

	switch node.GetNodeType() {
	case tree.NtElem:
		ret = tok.(xml.StartElement).Name.Space
	case tree.NtAttr:
		ret = tok.(xml.Attr).Name.Space
	}

	return tree.String(ret), nil
}

func name(c tree.Ctx, args ...tree.Result) (tree.Result, error) {
	var n tree.NodeSet
	ok := true
	if len(args) == 1 {
		n, ok = args[0].(tree.NodeSet)
	} else {
		n = c.NodeSet
	}
	if !ok {
		return nil, fmt.Errorf("Cannot convert object to a node-set")
	}

	ret := ""
	if len(n) == 0 {
		return tree.String(ret), nil
	}
	node := n[0]

	switch node.GetNodeType() {
	case tree.NtElem:
		t := node.GetToken().(xml.StartElement)
		space := ""

		if t.Name.Space != "" {
			space = fmt.Sprintf("{%s}", t.Name.Space)
		}

		ret = fmt.Sprintf("%s%s", space, t.Name.Local)
	case tree.NtAttr:
		t := node.GetToken().(xml.Attr)
		space := ""

		if t.Name.Space != "" {
			space = fmt.Sprintf("{%s}", t.Name.Space)
		}

		ret = fmt.Sprintf("%s%s", space, t.Name.Local)
	case tree.NtPi:
		ret = fmt.Sprintf("%s", node.GetToken().(xml.ProcInst).Target)
	}

	return tree.String(ret), nil
}
