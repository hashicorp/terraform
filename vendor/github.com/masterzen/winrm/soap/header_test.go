package soap

import (
	"github.com/masterzen/simplexml/dom"
	. "gopkg.in/check.v1"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func initDocument() (h *SoapHeader) {
	doc := dom.CreateDocument()
	doc.PrettyPrint = true
	e := dom.CreateElement("Envelope")
	doc.SetRoot(e)
	AddUsualNamespaces(e)
	NS_SOAP_ENV.SetTo(e)
	h = &SoapHeader{message: &SoapMessage{document: doc, envelope: e}}
	return
}

func (s *MySuite) TestHeaderBuild(c *C) {
	h := initDocument()
	msg := h.To("http://winrm:5985/wsman").ReplyTo("http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous").MaxEnvelopeSize(153600).Id("1-2-3-4").Locale("en_US").Timeout("PT60S").Build()

	expected := `<?xml version="1.0" encoding="utf-8" ?>
<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell" xmlns:w="http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd" xmlns:p="http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd">
  <env:Header>
    <a:To>http://winrm:5985/wsman</a:To>
    <a:ReplyTo>
      <a:Address mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/08/addressing/role/anonymous</a:Address>
    </a:ReplyTo>
    <w:MaxEnvelopeSize mustUnderstand="true">153600</w:MaxEnvelopeSize>
    <w:OperationTimeout>PT60S</w:OperationTimeout>
    <a:MessageID>1-2-3-4</a:MessageID>
    <w:Locale mustUnderstand="false" xml:lang="en_US"/>
    <p:DataLocale mustUnderstand="false" xml:lang="en_US"/>
  </env:Header>
</env:Envelope>
`

	c.Check(msg.String(), Equals, expected)
}
