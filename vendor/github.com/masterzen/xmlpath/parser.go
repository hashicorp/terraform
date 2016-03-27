package xmlpath

import (
	"encoding/xml"
	"io"
)

// Node is an item in an xml tree that was compiled to
// be processed via xml paths. A node may represent:
//
//     - An element in the xml document (<body>)
//     - An attribute of an element in the xml document (href="...")
//     - A comment in the xml document (<!--...-->)
//     - A processing instruction in the xml document (<?...?>)
//     - Some text within the xml document
//
type Node struct {
	kind nodeKind
	name xml.Name
	attr string
	text []byte

	nodes []Node
	pos   int
	end   int

	up   *Node
	down []*Node
}

type nodeKind int

const (
	anyNode nodeKind = iota
	startNode
	endNode
	attrNode
	textNode
	commentNode
	procInstNode
)

// String returns the string value of node.
//
// The string value of a node is:
//
//     - For element nodes, the concatenation of all text nodes within the element.
//     - For text nodes, the text itself.
//     - For attribute nodes, the attribute value.
//     - For comment nodes, the text within the comment delimiters.
//     - For processing instruction nodes, the content of the instruction.
//
func (node *Node) String() string {
	if node.kind == attrNode {
		return node.attr
	}
	return string(node.Bytes())
}

// Bytes returns the string value of node as a byte slice.
// See Node.String for a description of what the string value of a node is.
func (node *Node) Bytes() []byte {
	if node.kind == attrNode {
		return []byte(node.attr)
	}
	if node.kind != startNode {
		return node.text
	}
	var text []byte
	for i := node.pos; i < node.end; i++ {
		if node.nodes[i].kind == textNode {
			text = append(text, node.nodes[i].text...)
		}
	}
	return text
}

// equals returns whether the string value of node is equal to s,
// without allocating memory.
func (node *Node) equals(s string) bool {
	if node.kind == attrNode {
		return s == node.attr
	}
	if node.kind != startNode {
		if len(s) != len(node.text) {
			return false
		}
		for i := range s {
			if s[i] != node.text[i] {
				return false
			}
		}
		return true
	}
	si := 0
	for i := node.pos; i < node.end; i++ {
		if node.nodes[i].kind == textNode {
			for _, c := range node.nodes[i].text {
				if si > len(s) {
					return false
				}
				if s[si] != c {
					return false
				}
				si++
			}
		}
	}
	return si == len(s)
}

// Parse reads an xml document from r, parses it, and returns its root node.
func Parse(r io.Reader) (*Node, error) {
	return ParseDecoder(xml.NewDecoder(r))
}

// ParseHTML reads an HTML-like document from r, parses it, and returns
// its root node.
func ParseHTML(r io.Reader) (*Node, error) {
	d := xml.NewDecoder(r)
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	return ParseDecoder(d)
}

// ParseDecoder parses the xml document being decoded by d and returns
// its root node.
func ParseDecoder(d *xml.Decoder) (*Node, error) {
	var nodes []Node
	var text []byte

	// The root node.
	nodes = append(nodes, Node{kind: startNode})

	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := t.(type) {
		case xml.EndElement:
			nodes = append(nodes, Node{
				kind: endNode,
			})
		case xml.StartElement:
			nodes = append(nodes, Node{
				kind: startNode,
				name: t.Name,
			})
			for _, attr := range t.Attr {
				nodes = append(nodes, Node{
					kind: attrNode,
					name: attr.Name,
					attr: attr.Value,
				})
			}
		case xml.CharData:
			texti := len(text)
			text = append(text, t...)
			nodes = append(nodes, Node{
				kind: textNode,
				text: text[texti : texti+len(t)],
			})
		case xml.Comment:
			texti := len(text)
			text = append(text, t...)
			nodes = append(nodes, Node{
				kind: commentNode,
				text: text[texti : texti+len(t)],
			})
		case xml.ProcInst:
			texti := len(text)
			text = append(text, t.Inst...)
			nodes = append(nodes, Node{
				kind: procInstNode,
				name: xml.Name{Local: t.Target},
				text: text[texti : texti+len(t.Inst)],
			})
		}
	}

	// Close the root node.
	nodes = append(nodes, Node{kind: endNode})

	stack := make([]*Node, 0, len(nodes))
	downs := make([]*Node, len(nodes))
	downCount := 0

	for pos := range nodes {

		switch nodes[pos].kind {

		case startNode, attrNode, textNode, commentNode, procInstNode:
			node := &nodes[pos]
			node.nodes = nodes
			node.pos = pos
			if len(stack) > 0 {
				node.up = stack[len(stack)-1]
			}
			if node.kind == startNode {
				stack = append(stack, node)
			} else {
				node.end = pos + 1
			}

		case endNode:
			node := stack[len(stack)-1]
			node.end = pos
			stack = stack[:len(stack)-1]

			// Compute downs. Doing that here is what enables the
			// use of a slice of a contiguous pre-allocated block.
			node.down = downs[downCount:downCount]
			for i := node.pos + 1; i < node.end; i++ {
				if nodes[i].up == node {
					switch nodes[i].kind {
					case startNode, textNode, commentNode, procInstNode:
						node.down = append(node.down, &nodes[i])
						downCount++
					}
				}
			}
			if len(stack) == 0 {
				return node, nil
			}
		}
	}
	return nil, io.EOF
}
