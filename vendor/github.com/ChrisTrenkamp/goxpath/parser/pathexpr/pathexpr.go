package pathexpr

import "encoding/xml"

//PathExpr represents XPath step's.  xmltree.XMLTree uses it to find nodes.
type PathExpr struct {
	Name     xml.Name
	Axis     string
	NodeType string
	NS       map[string]string
}
