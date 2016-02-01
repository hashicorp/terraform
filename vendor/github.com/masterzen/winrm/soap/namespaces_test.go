package soap

import (
	"github.com/masterzen/simplexml/dom"
	"testing"
)

func TestAddUsualNamespaces(t *testing.T) {
	doc := dom.CreateDocument()
	root := dom.CreateElement("root")
	doc.SetRoot(root)
	AddUsualNamespaces(root)

	for ns := range root.DeclaredNamespaces() {
		found := false
		for ns2 := range MostUsed {
			if ns2 == ns {
				found = true
			}
		}
		if !found {
			t.Errorf("Test failed - Namespace %s not found", ns)
		}
	}

}

func TestSetTo(t *testing.T) {
	doc := dom.CreateDocument()
	root := dom.CreateElement("root")
	doc.SetRoot(root)
	NS_SOAP_ENV.SetTo(root)

	if root.String() != `<env:root xmlns:env="http://www.w3.org/2003/05/soap-envelope"/>` {
		t.Errorf("Test failed - root has not the correct NS: %s", root.String())
	}
}
