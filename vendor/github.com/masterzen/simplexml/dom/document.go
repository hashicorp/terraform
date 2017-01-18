package dom

import (
	"bytes"
	"fmt"
)

type Document struct {
	root        *Element
	PrettyPrint bool
	Indentation string
	DocType     bool
}

func CreateDocument() *Document {
	return &Document{PrettyPrint: false, Indentation: "  ", DocType: true}
}

func (doc *Document) SetRoot(node *Element) {
	node.parent = nil
	doc.root = node
}

func (doc *Document) String() string {
	var b bytes.Buffer
	if doc.DocType {
		fmt.Fprintln(&b, `<?xml version="1.0" encoding="utf-8" ?>`)
	}

	if doc.root != nil {
		doc.root.Bytes(&b, doc.PrettyPrint, doc.Indentation, 0)
	}

	return string(b.Bytes())
}
