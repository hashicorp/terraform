package dom

type Namespace struct {
	Prefix string
	Uri    string
}

func (ns *Namespace) SetTo(node *Element) {
	node.SetNamespace(ns.Prefix, ns.Uri)
}
