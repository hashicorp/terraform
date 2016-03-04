package soap

import (
	"github.com/masterzen/simplexml/dom"
	"strconv"
)

type HeaderOption struct {
	key   string
	value string
}

func NewHeaderOption(name string, value string) *HeaderOption {
	return &HeaderOption{key: name, value: value}
}

type SoapHeader struct {
	to              string
	replyTo         string
	maxEnvelopeSize string
	timeout         string
	locale          string
	id              string
	action          string
	shellId         string
	resourceURI     string
	options         []HeaderOption
	message         *SoapMessage
}

type HeaderBuilder interface {
	To(string) *SoapHeader
	ReplyTo(string) *SoapHeader
	MaxEnvelopeSize(int) *SoapHeader
	Timeout(string) *SoapHeader
	Locale(string) *SoapHeader
	Id(string) *SoapHeader
	Action(string) *SoapHeader
	ShellId(string) *SoapHeader
	resourceURI(string) *SoapHeader
	AddOption(*HeaderOption) *SoapHeader
	Options([]HeaderOption) *SoapHeader
	Build(*SoapMessage) *SoapMessage
}

func (self *SoapHeader) To(uri string) *SoapHeader {
	self.to = uri
	return self
}

func (self *SoapHeader) ReplyTo(uri string) *SoapHeader {
	self.replyTo = uri
	return self
}

func (self *SoapHeader) MaxEnvelopeSize(size int) *SoapHeader {
	self.maxEnvelopeSize = strconv.Itoa(size)
	return self
}

func (self *SoapHeader) Timeout(timeout string) *SoapHeader {
	self.timeout = timeout
	return self
}

func (self *SoapHeader) Id(id string) *SoapHeader {
	self.id = id
	return self
}

func (self *SoapHeader) Action(action string) *SoapHeader {
	self.action = action
	return self
}

func (self *SoapHeader) Locale(locale string) *SoapHeader {
	self.locale = locale
	return self
}

func (self *SoapHeader) ShellId(shellId string) *SoapHeader {
	self.shellId = shellId
	return self
}

func (self *SoapHeader) ResourceURI(resourceURI string) *SoapHeader {
	self.resourceURI = resourceURI
	return self
}

func (self *SoapHeader) AddOption(option *HeaderOption) *SoapHeader {
	self.options = append(self.options, *option)
	return self
}

func (self *SoapHeader) Options(options []HeaderOption) *SoapHeader {
	self.options = options
	return self
}

func (self *SoapHeader) Build() *SoapMessage {
	header := self.createElement(self.message.envelope, "Header", NS_SOAP_ENV)

	if self.to != "" {
		to := self.createElement(header, "To", NS_ADDRESSING)
		to.SetContent(self.to)
	}

	if self.replyTo != "" {
		replyTo := self.createElement(header, "ReplyTo", NS_ADDRESSING)
		a := self.createMUElement(replyTo, "Address", NS_ADDRESSING, true)
		a.SetContent(self.replyTo)
	}

	if self.maxEnvelopeSize != "" {
		envelope := self.createMUElement(header, "MaxEnvelopeSize", NS_WSMAN_DMTF, true)
		envelope.SetContent(self.maxEnvelopeSize)
	}

	if self.timeout != "" {
		timeout := self.createElement(header, "OperationTimeout", NS_WSMAN_DMTF)
		timeout.SetContent(self.timeout)
	}

	if self.id != "" {
		id := self.createElement(header, "MessageID", NS_ADDRESSING)
		id.SetContent(self.id)
	}

	if self.locale != "" {
		locale := self.createMUElement(header, "Locale", NS_WSMAN_DMTF, false)
		locale.SetAttr("xml:lang", self.locale)
		datalocale := self.createMUElement(header, "DataLocale", NS_WSMAN_MSFT, false)
		datalocale.SetAttr("xml:lang", self.locale)
	}

	if self.action != "" {
		action := self.createMUElement(header, "Action", NS_ADDRESSING, true)
		action.SetContent(self.action)
	}

	if self.shellId != "" {
		selectorSet := self.createElement(header, "SelectorSet", NS_WSMAN_DMTF)
		selector := self.createElement(selectorSet, "Selector", NS_WSMAN_DMTF)
		selector.SetAttr("Name", "ShellId")
		selector.SetContent(self.shellId)
	}

	if self.resourceURI != "" {
		resource := self.createMUElement(header, "ResourceURI", NS_WSMAN_DMTF, true)
		resource.SetContent(self.resourceURI)
	}

	if len(self.options) > 0 {
		set := self.createElement(header, "OptionSet", NS_WSMAN_DMTF)
		for _, option := range self.options {
			e := self.createElement(set, "Option", NS_WSMAN_DMTF)
			e.SetAttr("Name", option.key)
			e.SetContent(option.value)
		}
	}

	return self.message
}

func (self *SoapHeader) createElement(parent *dom.Element, name string, ns dom.Namespace) (element *dom.Element) {
	element = dom.CreateElement(name)
	parent.AddChild(element)
	ns.SetTo(element)
	return
}

func (self *SoapHeader) createMUElement(parent *dom.Element, name string, ns dom.Namespace, mustUnderstand bool) (element *dom.Element) {
	element = self.createElement(parent, name, ns)
	value := "false"
	if mustUnderstand {
		value = "true"
	}
	element.SetAttr("mustUnderstand", value)
	return
}
