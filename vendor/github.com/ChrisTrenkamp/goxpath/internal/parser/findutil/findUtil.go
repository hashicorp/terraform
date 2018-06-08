package findutil

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/internal/parser/pathexpr"
	"github.com/ChrisTrenkamp/goxpath/internal/xconst"
	"github.com/ChrisTrenkamp/goxpath/tree"
)

const (
	wildcard = "*"
)

type findFunc func(tree.Node, *pathexpr.PathExpr, *[]tree.Node)

var findMap = map[string]findFunc{
	xconst.AxisAncestor:         findAncestor,
	xconst.AxisAncestorOrSelf:   findAncestorOrSelf,
	xconst.AxisAttribute:        findAttribute,
	xconst.AxisChild:            findChild,
	xconst.AxisDescendent:       findDescendent,
	xconst.AxisDescendentOrSelf: findDescendentOrSelf,
	xconst.AxisFollowing:        findFollowing,
	xconst.AxisFollowingSibling: findFollowingSibling,
	xconst.AxisNamespace:        findNamespace,
	xconst.AxisParent:           findParent,
	xconst.AxisPreceding:        findPreceding,
	xconst.AxisPrecedingSibling: findPrecedingSibling,
	xconst.AxisSelf:             findSelf,
}

//Find finds nodes based on the pathexpr.PathExpr
func Find(x tree.Node, p pathexpr.PathExpr) []tree.Node {
	ret := []tree.Node{}

	if p.Axis == "" {
		findChild(x, &p, &ret)
		return ret
	}

	f := findMap[p.Axis]
	f(x, &p, &ret)

	return ret
}

func findAncestor(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() == tree.NtRoot {
		return
	}

	addNode(x.GetParent(), p, ret)
	findAncestor(x.GetParent(), p, ret)
}

func findAncestorOrSelf(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	findSelf(x, p, ret)
	findAncestor(x, p, ret)
}

func findAttribute(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if ele, ok := x.(tree.Elem); ok {
		for _, i := range ele.GetAttrs() {
			addNode(i, p, ret)
		}
	}
}

func findChild(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if ele, ok := x.(tree.Elem); ok {
		ch := ele.GetChildren()
		for i := range ch {
			addNode(ch[i], p, ret)
		}
	}
}

func findDescendent(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if ele, ok := x.(tree.Elem); ok {
		ch := ele.GetChildren()
		for i := range ch {
			addNode(ch[i], p, ret)
			findDescendent(ch[i], p, ret)
		}
	}
}

func findDescendentOrSelf(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	findSelf(x, p, ret)
	findDescendent(x, p, ret)
}

func findFollowing(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() == tree.NtRoot {
		return
	}
	par := x.GetParent()
	ch := par.GetChildren()
	i := 0
	for x != ch[i] {
		i++
	}
	i++
	for i < len(ch) {
		findDescendentOrSelf(ch[i], p, ret)
		i++
	}
	findFollowing(par, p, ret)
}

func findFollowingSibling(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() == tree.NtRoot {
		return
	}
	par := x.GetParent()
	ch := par.GetChildren()
	i := 0
	for x != ch[i] {
		i++
	}
	i++
	for i < len(ch) {
		findSelf(ch[i], p, ret)
		i++
	}
}

func findNamespace(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if ele, ok := x.(tree.NSElem); ok {
		ns := tree.BuildNS(ele)
		for _, i := range ns {
			addNode(i, p, ret)
		}
	}
}

func findParent(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() != tree.NtRoot {
		addNode(x.GetParent(), p, ret)
	}
}

func findPreceding(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() == tree.NtRoot {
		return
	}
	par := x.GetParent()
	ch := par.GetChildren()
	i := len(ch) - 1
	for x != ch[i] {
		i--
	}
	i--
	for i >= 0 {
		findDescendentOrSelf(ch[i], p, ret)
		i--
	}
	findPreceding(par, p, ret)
}

func findPrecedingSibling(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	if x.GetNodeType() == tree.NtRoot {
		return
	}
	par := x.GetParent()
	ch := par.GetChildren()
	i := len(ch) - 1
	for x != ch[i] {
		i--
	}
	i--
	for i >= 0 {
		findSelf(ch[i], p, ret)
		i--
	}
}

func findSelf(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	addNode(x, p, ret)
}

func addNode(x tree.Node, p *pathexpr.PathExpr, ret *[]tree.Node) {
	add := false
	tok := x.GetToken()

	switch x.GetNodeType() {
	case tree.NtAttr:
		add = evalAttr(p, tok.(xml.Attr))
	case tree.NtChd:
		add = evalChd(p)
	case tree.NtComm:
		add = evalComm(p)
	case tree.NtElem, tree.NtRoot:
		add = evalEle(p, tok.(xml.StartElement))
	case tree.NtNs:
		add = evalNS(p, tok.(xml.Attr))
	case tree.NtPi:
		add = evalPI(p)
	}

	if add {
		*ret = append(*ret, x)
	}
}

func evalAttr(p *pathexpr.PathExpr, a xml.Attr) bool {
	if p.NodeType == "" {
		if p.Name.Space != wildcard {
			if a.Name.Space != p.NS[p.Name.Space] {
				return false
			}
		}

		if p.Name.Local == wildcard && p.Axis == xconst.AxisAttribute {
			return true
		}

		if p.Name.Local == a.Name.Local {
			return true
		}
	} else {
		if p.NodeType == xconst.NodeTypeNode {
			return true
		}
	}

	return false
}

func evalChd(p *pathexpr.PathExpr) bool {
	if p.NodeType == xconst.NodeTypeText || p.NodeType == xconst.NodeTypeNode {
		return true
	}

	return false
}

func evalComm(p *pathexpr.PathExpr) bool {
	if p.NodeType == xconst.NodeTypeComment || p.NodeType == xconst.NodeTypeNode {
		return true
	}

	return false
}

func evalEle(p *pathexpr.PathExpr, ele xml.StartElement) bool {
	if p.NodeType == "" {
		return checkNameAndSpace(p, ele)
	}

	if p.NodeType == xconst.NodeTypeNode {
		return true
	}

	return false
}

func checkNameAndSpace(p *pathexpr.PathExpr, ele xml.StartElement) bool {
	if p.Name.Local == wildcard && p.Name.Space == "" {
		return true
	}

	if p.Name.Space != wildcard && ele.Name.Space != p.NS[p.Name.Space] {
		return false
	}

	if p.Name.Local == wildcard && p.Axis != xconst.AxisAttribute && p.Axis != xconst.AxisNamespace {
		return true
	}

	if p.Name.Local == ele.Name.Local {
		return true
	}

	return false
}

func evalNS(p *pathexpr.PathExpr, ns xml.Attr) bool {
	if p.NodeType == "" {
		if p.Name.Space != "" && p.Name.Space != wildcard {
			return false
		}

		if p.Name.Local == wildcard && p.Axis == xconst.AxisNamespace {
			return true
		}

		if p.Name.Local == ns.Name.Local {
			return true
		}
	} else {
		if p.NodeType == xconst.NodeTypeNode {
			return true
		}
	}

	return false
}

func evalPI(p *pathexpr.PathExpr) bool {
	if p.NodeType == xconst.NodeTypeProcInst || p.NodeType == xconst.NodeTypeNode {
		return true
	}

	return false
}
