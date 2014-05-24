package digraph

import (
	"fmt"
	"strings"
)

// BasicNode is a digraph Node that has a name and out edges
type BasicNode struct {
	Name      string
	NodeEdges []Edge
}

func (b *BasicNode) Edges() []Edge {
	return b.NodeEdges
}

func (b *BasicNode) AddEdge(edge Edge) {
	b.NodeEdges = append(b.NodeEdges, edge)
}

func (b *BasicNode) String() string {
	if b.Name == "" {
		return "Node"
	}
	return fmt.Sprintf("%v", b.Name)
}

// BasicEdge is a digraph Edge that has a name, head and tail
type BasicEdge struct {
	Name     string
	EdgeHead *BasicNode
	EdgeTail *BasicNode
}

func (b *BasicEdge) Head() Node {
	return b.EdgeHead
}

// Tail returns the end point of the Edge
func (b *BasicEdge) Tail() Node {
	return b.EdgeTail
}

func (b *BasicEdge) String() string {
	if b.Name == "" {
		return "Edge"
	}
	return fmt.Sprintf("%v", b.Name)
}

// ParseBasic is used to parse a string in the format of:
// a -> b ; edge name
// b -> c
// Into a series of basic node and basic edges
func ParseBasic(s string) map[string]*BasicNode {
	lines := strings.Split(s, "\n")
	nodes := make(map[string]*BasicNode)
	for _, line := range lines {
		var edgeName string
		if idx := strings.Index(line, ";"); idx >= 0 {
			edgeName = strings.Trim(line[idx+1:], " \t\r\n")
			line = line[:idx]
		}
		parts := strings.SplitN(line, "->", 2)
		if len(parts) != 2 {
			continue
		}
		head_name := strings.Trim(parts[0], " \t\r\n")
		tail_name := strings.Trim(parts[1], " \t\r\n")
		head := nodes[head_name]
		if head == nil {
			head = &BasicNode{Name: head_name}
			nodes[head_name] = head
		}
		tail := nodes[tail_name]
		if tail == nil {
			tail = &BasicNode{Name: tail_name}
			nodes[tail_name] = tail
		}
		edge := &BasicEdge{
			Name:     edgeName,
			EdgeHead: head,
			EdgeTail: tail,
		}
		head.AddEdge(edge)
	}
	return nodes
}
