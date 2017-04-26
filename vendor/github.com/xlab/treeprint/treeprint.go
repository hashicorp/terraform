// Package treeprint provides a simple ASCII tree composing tool.
package treeprint

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

type Value interface{}
type MetaValue interface{}

// Tree represents a tree structure with leaf-nodes and branch-nodes.
type Tree interface {
	// AddNode adds a new node to a branch.
	AddNode(v Value) Tree
	// AddMetaNode adds a new node with meta value provided to a branch.
	AddMetaNode(meta MetaValue, v Value) Tree
	// AddBranch adds a new branch node (a level deeper).
	AddBranch(v Value) Tree
	// AddMetaBranch adds a new branch node (a level deeper) with meta value provided.
	AddMetaBranch(meta MetaValue, v Value) Tree
	// Branch converts a leaf-node to a branch-node,
	// applying this on a branch-node does no effect.
	Branch() Tree
	// FindByMeta finds a node whose meta value matches the provided one by reflect.DeepEqual,
	// returns nil if not found.
	FindByMeta(meta MetaValue) Tree
	// FindByValue finds a node whose value matches the provided one by reflect.DeepEqual,
	// returns nil if not found.
	FindByValue(value Value) Tree
	// String renders the tree or subtree as a string.
	String() string
	// Bytes renders the tree or subtree as byteslice.
	Bytes() []byte
}

type node struct {
	Root  *node
	Meta  MetaValue
	Value Value
	Nodes []*node
}

func (n *node) AddNode(v Value) Tree {
	n.Nodes = append(n.Nodes, &node{
		Root:  n,
		Value: v,
	})
	if n.Root != nil {
		return n.Root
	}
	return n
}

func (n *node) AddMetaNode(meta MetaValue, v Value) Tree {
	n.Nodes = append(n.Nodes, &node{
		Root:  n,
		Meta:  meta,
		Value: v,
	})
	if n.Root != nil {
		return n.Root
	}
	return n
}

func (n *node) AddBranch(v Value) Tree {
	branch := &node{
		Value: v,
	}
	n.Nodes = append(n.Nodes, branch)
	return branch
}

func (n *node) AddMetaBranch(meta MetaValue, v Value) Tree {
	branch := &node{
		Meta:  meta,
		Value: v,
	}
	n.Nodes = append(n.Nodes, branch)
	return branch
}

func (n *node) Branch() Tree {
	n.Root = nil
	return n
}

func (n *node) FindByMeta(meta MetaValue) Tree {
	for _, node := range n.Nodes {
		if reflect.DeepEqual(node.Meta, meta) {
			return node
		}
		if v := node.FindByMeta(meta); v != nil {
			return v
		}
	}
	return nil
}

func (n *node) FindByValue(value Value) Tree {
	for _, node := range n.Nodes {
		if reflect.DeepEqual(node.Value, value) {
			return node
		}
		if v := node.FindByMeta(value); v != nil {
			return v
		}
	}
	return nil
}

func (n *node) Bytes() []byte {
	buf := new(bytes.Buffer)
	level := 0
	levelEnded := make(map[int]bool)
	if n.Root == nil {
		buf.WriteString(string(EdgeTypeStart))
		buf.WriteByte('\n')
	} else {
		edge := EdgeTypeMid
		if len(n.Nodes) == 0 {
			edge = EdgeTypeEnd
			levelEnded[level] = true
		}
		printValues(buf, 0, levelEnded, edge, n.Meta, n.Value)
	}
	if len(n.Nodes) > 0 {
		printNodes(buf, level, levelEnded, n.Nodes)
	}
	return buf.Bytes()
}

func (n *node) String() string {
	return string(n.Bytes())
}

func printNodes(wr io.Writer,
	level int, levelEnded map[int]bool, nodes []*node) {

	for i, node := range nodes {
		edge := EdgeTypeMid
		if i == len(nodes)-1 {
			levelEnded[level] = true
			edge = EdgeTypeEnd
		}
		printValues(wr, level, levelEnded, edge, node.Meta, node.Value)
		if len(node.Nodes) > 0 {
			printNodes(wr, level+1, levelEnded, node.Nodes)
		}
	}
}

func printValues(wr io.Writer,
	level int, levelEnded map[int]bool, edge EdgeType, meta MetaValue, val Value) {

	for i := 0; i < level; i++ {
		if levelEnded[i] {
			fmt.Fprint(wr, "    ")
			continue
		}
		fmt.Fprintf(wr, "%s   ", EdgeTypeLink)
	}
	if meta != nil {
		fmt.Fprintf(wr, "%s [%v]  %v\n", edge, meta, val)
		return
	}
	fmt.Fprintf(wr, "%s %v\n", edge, val)
}

type EdgeType string

const (
	EdgeTypeStart EdgeType = "."
	EdgeTypeLink  EdgeType = "│"
	EdgeTypeMid   EdgeType = "├──"
	EdgeTypeEnd   EdgeType = "└──"
)

func New() Tree {
	return &node{}
}
