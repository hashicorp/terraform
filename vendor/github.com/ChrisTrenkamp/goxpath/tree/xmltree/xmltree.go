package xmltree

import (
	"encoding/xml"
	"io"

	"golang.org/x/net/html/charset"

	"github.com/ChrisTrenkamp/goxpath/tree"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree/xmlbuilder"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree/xmlele"
)

//ParseOptions is a set of methods and function pointers that alter
//the way the XML decoder works and the Node types that are created.
//Options that are not set will default to what is set in internal/defoverride.go
type ParseOptions struct {
	Strict  bool
	XMLRoot func() xmlbuilder.XMLBuilder
}

//DirectiveParser is an optional interface extended from XMLBuilder that handles
//XML directives.
type DirectiveParser interface {
	xmlbuilder.XMLBuilder
	Directive(xml.Directive, *xml.Decoder)
}

//ParseSettings is a function for setting the ParseOptions you want when
//parsing an XML tree.
type ParseSettings func(s *ParseOptions)

//MustParseXML is like ParseXML, but panics instead of returning an error.
func MustParseXML(r io.Reader, op ...ParseSettings) tree.Node {
	ret, err := ParseXML(r, op...)

	if err != nil {
		panic(err)
	}

	return ret
}

//ParseXML creates an XMLTree structure from an io.Reader.
func ParseXML(r io.Reader, op ...ParseSettings) (tree.Node, error) {
	ov := ParseOptions{
		Strict:  true,
		XMLRoot: xmlele.Root,
	}
	for _, i := range op {
		i(&ov)
	}

	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = ov.Strict

	ordrPos := 1
	xmlTree := ov.XMLRoot()

	t, err := dec.Token()

	if err != nil {
		return nil, err
	}

	if head, ok := t.(xml.ProcInst); ok && head.Target == "xml" {
		t, err = dec.Token()
	}

	opts := xmlbuilder.BuilderOpts{
		Dec: dec,
	}

	for err == nil {
		switch xt := t.(type) {
		case xml.StartElement:
			setEle(&opts, xmlTree, xt, &ordrPos)
			xmlTree = xmlTree.CreateNode(&opts)
		case xml.CharData:
			setNode(&opts, xmlTree, xt, tree.NtChd, &ordrPos)
			xmlTree = xmlTree.CreateNode(&opts)
		case xml.Comment:
			setNode(&opts, xmlTree, xt, tree.NtComm, &ordrPos)
			xmlTree = xmlTree.CreateNode(&opts)
		case xml.ProcInst:
			setNode(&opts, xmlTree, xt, tree.NtPi, &ordrPos)
			xmlTree = xmlTree.CreateNode(&opts)
		case xml.EndElement:
			xmlTree = xmlTree.EndElem()
		case xml.Directive:
			if dp, ok := xmlTree.(DirectiveParser); ok {
				dp.Directive(xt.Copy(), dec)
			}
		}

		t, err = dec.Token()
	}

	if err == io.EOF {
		err = nil
	}

	return xmlTree, err
}

func setEle(opts *xmlbuilder.BuilderOpts, xmlTree xmlbuilder.XMLBuilder, ele xml.StartElement, ordrPos *int) {
	opts.NodePos = *ordrPos
	opts.Tok = ele
	opts.Attrs = opts.Attrs[0:0:cap(opts.Attrs)]
	opts.NS = make(map[xml.Name]string)
	opts.NodeType = tree.NtElem

	for i := range ele.Attr {
		attr := ele.Attr[i].Name
		val := ele.Attr[i].Value

		if (attr.Local == "xmlns" && attr.Space == "") || attr.Space == "xmlns" {
			opts.NS[attr] = val
		} else {
			opts.Attrs = append(opts.Attrs, &ele.Attr[i])
		}
	}

	if nstree, ok := xmlTree.(tree.NSElem); ok {
		ns := make(map[xml.Name]string)

		for _, i := range tree.BuildNS(nstree) {
			ns[i.Name] = i.Value
		}

		for k, v := range opts.NS {
			ns[k] = v
		}

		if ns[xml.Name{Local: "xmlns"}] == "" {
			delete(ns, xml.Name{Local: "xmlns"})
		}

		for k, v := range ns {
			opts.NS[k] = v
		}

		if xmlTree.GetNodeType() == tree.NtRoot {
			opts.NS[xml.Name{Space: "xmlns", Local: "xml"}] = tree.XMLSpace
		}
	}

	opts.AttrStartPos = len(opts.NS) + len(opts.Attrs) + *ordrPos
	*ordrPos = opts.AttrStartPos + 1
}

func setNode(opts *xmlbuilder.BuilderOpts, xmlTree xmlbuilder.XMLBuilder, tok xml.Token, nt tree.NodeType, ordrPos *int) {
	opts.Tok = xml.CopyToken(tok)
	opts.NodeType = nt
	opts.NodePos = *ordrPos
	*ordrPos++
}
