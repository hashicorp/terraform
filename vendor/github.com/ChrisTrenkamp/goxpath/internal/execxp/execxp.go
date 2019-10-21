package execxp

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/parser"
	"github.com/ChrisTrenkamp/goxpath/tree"
)

//Exec executes the XPath expression, xp, against the tree, t, with the
//namespace mappings, ns.
func Exec(n *parser.Node, t tree.Node, ns map[string]string, fns map[xml.Name]tree.Wrap, v map[string]tree.Result) (tree.Result, error) {
	f := xpFilt{
		t:         t,
		ns:        ns,
		ctx:       tree.NodeSet{t},
		fns:       fns,
		variables: v,
	}

	return exec(&f, n)
}

func exec(f *xpFilt, n *parser.Node) (tree.Result, error) {
	err := xfExec(f, n)
	return f.ctx, err
}
