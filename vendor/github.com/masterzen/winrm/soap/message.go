package soap

import (
	"github.com/masterzen/simplexml/dom"
)

type SoapMessage struct {
	document *dom.Document
	envelope *dom.Element
	header   *SoapHeader
	body     *dom.Element
}

type MessageBuilder interface {
	SetBody(*dom.Element)
	NewBody() *dom.Element
	CreateElement(*dom.Element, string, dom.Namespace) *dom.Element
	CreateBodyElement(string, dom.Namespace) *dom.Element
	Header() *SoapHeader
	Doc() *dom.Document
	Free()
	String() string
}

func NewMessage() (message *SoapMessage) {
	doc := dom.CreateDocument()
	e := dom.CreateElement("Envelope")
	doc.SetRoot(e)
	AddUsualNamespaces(e)
	DOM_NS_SOAP_ENV.SetTo(e)

	message = &SoapMessage{document: doc, envelope: e}
	return
}

func (message *SoapMessage) NewBody() (body *dom.Element) {
	body = dom.CreateElement("Body")
	message.envelope.AddChild(body)
	DOM_NS_SOAP_ENV.SetTo(body)
	return
}

func (message *SoapMessage) String() string {
	return message.document.String()
}

func (message *SoapMessage) Doc() *dom.Document {
	return message.document
}

func (message *SoapMessage) Free() {
}

func (message *SoapMessage) CreateElement(parent *dom.Element, name string, ns dom.Namespace) (element *dom.Element) {
	element = dom.CreateElement(name)
	parent.AddChild(element)
	ns.SetTo(element)
	return
}

func (message *SoapMessage) CreateBodyElement(name string, ns dom.Namespace) (element *dom.Element) {
	if message.body == nil {
		message.body = message.NewBody()
	}
	return message.CreateElement(message.body, name, ns)
}

func (message *SoapMessage) Header() *SoapHeader {
	if message.header == nil {
		message.header = &SoapHeader{message: message}
	}
	return message.header
}
