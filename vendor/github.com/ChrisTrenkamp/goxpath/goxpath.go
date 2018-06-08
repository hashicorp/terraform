package goxpath

import (
	"encoding/xml"
	"fmt"

	"github.com/ChrisTrenkamp/goxpath/internal/execxp"
	"github.com/ChrisTrenkamp/goxpath/internal/parser"
	"github.com/ChrisTrenkamp/goxpath/tree"
)

//Opts defines namespace mappings and custom functions for XPath expressions.
type Opts struct {
	NS    map[string]string
	Funcs map[xml.Name]tree.Wrap
	Vars  map[string]tree.Result
}

//FuncOpts is a function wrapper for Opts.
type FuncOpts func(*Opts)

//XPathExec is the XPath executor, compiled from an XPath string
type XPathExec struct {
	n *parser.Node
}

//Parse parses the XPath expression, xp, returning an XPath executor.
func Parse(xp string) (XPathExec, error) {
	n, err := parser.Parse(xp)
	return XPathExec{n: n}, err
}

//MustParse is like Parse, but panics instead of returning an error.
func MustParse(xp string) XPathExec {
	ret, err := Parse(xp)
	if err != nil {
		panic(err)
	}
	return ret
}

//Exec executes the XPath expression, xp, against the tree, t, with the
//namespace mappings, ns, and returns the result as a stringer.
func (xp XPathExec) Exec(t tree.Node, opts ...FuncOpts) (tree.Result, error) {
	o := &Opts{
		NS:    make(map[string]string),
		Funcs: make(map[xml.Name]tree.Wrap),
		Vars:  make(map[string]tree.Result),
	}
	for _, i := range opts {
		i(o)
	}
	return execxp.Exec(xp.n, t, o.NS, o.Funcs, o.Vars)
}

//ExecBool is like Exec, except it will attempt to convert the result to its boolean value.
func (xp XPathExec) ExecBool(t tree.Node, opts ...FuncOpts) (bool, error) {
	res, err := xp.Exec(t, opts...)
	if err != nil {
		return false, err
	}

	b, ok := res.(tree.IsBool)
	if !ok {
		return false, fmt.Errorf("Cannot convert result to a boolean")
	}

	return bool(b.Bool()), nil
}

//ExecNum is like Exec, except it will attempt to convert the result to its number value.
func (xp XPathExec) ExecNum(t tree.Node, opts ...FuncOpts) (float64, error) {
	res, err := xp.Exec(t, opts...)
	if err != nil {
		return 0, err
	}

	n, ok := res.(tree.IsNum)
	if !ok {
		return 0, fmt.Errorf("Cannot convert result to a number")
	}

	return float64(n.Num()), nil
}

//ExecNode is like Exec, except it will attempt to return the result as a node-set.
func (xp XPathExec) ExecNode(t tree.Node, opts ...FuncOpts) (tree.NodeSet, error) {
	res, err := xp.Exec(t, opts...)
	if err != nil {
		return nil, err
	}

	n, ok := res.(tree.NodeSet)
	if !ok {
		return nil, fmt.Errorf("Cannot convert result to a node-set")
	}

	return n, nil
}

//MustExec is like Exec, but panics instead of returning an error.
func (xp XPathExec) MustExec(t tree.Node, opts ...FuncOpts) tree.Result {
	res, err := xp.Exec(t, opts...)
	if err != nil {
		panic(err)
	}
	return res
}

//ParseExec parses the XPath string, xpstr, and runs Exec.
func ParseExec(xpstr string, t tree.Node, opts ...FuncOpts) (tree.Result, error) {
	xp, err := Parse(xpstr)
	if err != nil {
		return nil, err
	}
	return xp.Exec(t, opts...)
}
