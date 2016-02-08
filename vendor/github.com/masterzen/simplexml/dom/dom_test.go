package dom

import (
	. "launchpad.net/gocheck"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DomSuite struct{}

var _ = Suite(&DomSuite{})

func (s *DomSuite) TestEmptyDocument(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	c.Assert(doc.String(), Equals, "<?xml version=\"1.0\" encoding=\"utf-8\" ?>\n")
}

func (s *DomSuite) TestOneEmptyNode(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	doc.SetRoot(root)
	c.Assert(doc.String(), Equals, "<?xml version=\"1.0\" encoding=\"utf-8\" ?>\n<root/>\n")
}

func (s *DomSuite) TestMoreNodes(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	root.AddChild(node1)
	subnode := CreateElement("sub")
	node1.AddChild(subnode)
	node2 := CreateElement("node2")
	root.AddChild(node2)
	doc.SetRoot(root)
	
	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1>
    <sub/>
  </node1>
  <node2/>
</root>
`
	
	c.Assert(doc.String(), Equals, expected)
}

func (s *DomSuite) TestAttributes(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	node1.SetAttr("attr1", "pouet")
	root.AddChild(node1)
	doc.SetRoot(root)
	
	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1 attr1="pouet"/>
</root>
`
	c.Assert(doc.String(), Equals, expected)
}

func (s *DomSuite) TestContent(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	node1.SetContent("this is a text content")
	root.AddChild(node1)
	doc.SetRoot(root)
	
	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1>this is a text content</node1>
</root>
`
	c.Assert(doc.String(), Equals, expected)
}

func (s *DomSuite) TestNamespace(c *C) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	root.DeclareNamespace(Namespace { Prefix: "a", Uri: "http://schemas.xmlsoap.org/ws/2004/08/addressing"})
	node1 := CreateElement("node1")
	root.AddChild(node1)
	node1.SetNamespace("a", "http://schemas.xmlsoap.org/ws/2004/08/addressing")
	node1.SetContent("this is a text content")
	doc.SetRoot(root)
	
	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
  <a:node1>this is a text content</a:node1>
</root>
`
	c.Assert(doc.String(), Equals, expected)
}

