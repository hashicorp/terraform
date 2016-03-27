package dom

import (
	"encoding/xml"
	"fmt"
	"bytes"
)

type Attr struct {
	Name  xml.Name // Attribute namespace and name.
	Value string   // Attribute value.
}

type Element struct {
	name xml.Name
	children []*Element
	parent *Element
	content string
	attributes []*Attr
	namespaces []*Namespace
	document *Document
}

func CreateElement(n string) *Element {
	element := &Element { name: xml.Name { Local: n } }
	element.children = make([]*Element, 0, 5)
	element.attributes = make([]*Attr, 0, 10)
	element.namespaces  = make([]*Namespace, 0, 10)
	return element
}

func (node *Element) AddChild(child *Element) *Element {
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	child.SetParent(node)
	node.children = append(node.children, child)
	return node
}

func (node *Element) RemoveChild(child *Element) *Element {
	p := -1
	for i, v := range node.children {
		if v == child {
			p = i
			break
		}
	}

	if p == -1 {
		return node
	}

	copy(node.children[p:], node.children[p+1:])
	node.children = node.children[0 : len(node.children)-1]
	child.parent = nil
	return node
}

func (node *Element) SetAttr(name string, value string) *Element {
	// namespaces?
	attr := &Attr{ Name: xml.Name { Local: name }, Value: value }
	node.attributes = append(node.attributes, attr)
	return node
}

func (node *Element) SetParent(parent *Element) *Element {
	node.parent = parent
	return node
} 

func (node *Element) SetContent(content string) *Element {
	node.content = content
	return node
} 

// Add a namespace declaration to this node
func (node *Element) DeclareNamespace(ns Namespace) *Element {
	// check if we already have it
	prefix := node.namespacePrefix(ns.Uri)
	if  prefix == ns.Prefix {
		return node
	}
	// add it
	node.namespaces = append(node.namespaces, &ns)
	return node
}

func (node *Element) DeclaredNamespaces() []*Namespace {
	return node.namespaces
}

func (node *Element) SetNamespace(prefix string, uri string) {
	resolved := node.namespacePrefix(uri)
	if resolved == "" {
		// we couldn't find the namespace, let's declare it at this node
		node.namespaces = append(node.namespaces, &Namespace { Prefix: prefix, Uri: uri })
	}
	node.name.Space = uri
}

func (node *Element) Bytes(out *bytes.Buffer, indent bool, indentType string, level int) {
	empty := len(node.children) == 0 && node.content == ""
	content := node.content != ""
//	children := len(node.children) > 0
//	ns := len(node.namespaces) > 0
//	attrs := len(node.attributes) > 0
	
	indentStr := ""
	nextLine := ""
	if indent {
		nextLine = "\n"
		for i := 0; i < level; i++ {
	    	indentStr += indentType
		}
	}
	
	if node.name.Local != "" {
		if len(node.name.Space) > 0 {
			// first find if ns has been declared, otherwise
			prefix := node.namespacePrefix(node.name.Space)
			fmt.Fprintf(out, "%s<%s:%s", indentStr, prefix, node.name.Local)
		} else {
			fmt.Fprintf(out, "%s<%s", indentStr, node.name.Local)
		}
	}
	
	// declared namespaces
	for _, v := range node.namespaces {
		prefix := node.namespacePrefix(v.Uri)
		fmt.Fprintf(out, ` xmlns:%s="%s"`, prefix, v.Uri)
	}

	// attributes
	for _, v := range node.attributes {
		if len(v.Name.Space) > 0 {
			prefix := node.namespacePrefix(v.Name.Space)
			fmt.Fprintf(out, ` %s:%s="%s"`, prefix, v.Name.Local, v.Value)
		} else {
			fmt.Fprintf(out, ` %s="%s"`, v.Name.Local, v.Value)
		}
	}
	
	// close tag
	if empty {
		fmt.Fprintf(out, "/>%s", nextLine)
	} else {
		if content {
			out.WriteRune('>')
		} else {
			fmt.Fprintf(out, ">%s", nextLine)			
		}
	}
	
	if len(node.children) > 0 {
		for _, child := range node.children {
			child.Bytes(out, indent, indentType, level + 1)
		}
	} else if node.content != "" {
		//val := []byte(node.content)
		//xml.EscapeText(out, val)
		out.WriteString(node.content)
	}
	
	if !empty && len(node.name.Local) > 0 {
		var indentation string
		if content {
			indentation = ""
		} else {
			indentation = indentStr
		}
		if len(node.name.Space) > 0 {
			prefix := node.namespacePrefix(node.name.Space)
			fmt.Fprintf(out, "%s</%s:%s>\n", indentation, prefix, node.name.Local)
		} else {
			fmt.Fprintf(out, "%s</%s>\n", indentation, node.name.Local)
		}
	}
}

// Finds the prefix of the given namespace if it has been declared
// in this node or in one of its parent
func (node *Element) namespacePrefix(uri string) string {
	for _, ns := range node.namespaces {
		if ns.Uri == uri {
			return ns.Prefix
		}
	}
	if node.parent == nil {
		return ""
	}
	return node.parent.namespacePrefix(uri)
}


func (node *Element) String() string {
	var b bytes.Buffer
	node.Bytes(&b, false, "", 0)
	return string(b.Bytes())
}
