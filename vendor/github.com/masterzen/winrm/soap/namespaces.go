package soap

import (
	"github.com/masterzen/simplexml/dom"
	"github.com/masterzen/xmlpath"
)

var (
	NS_SOAP_ENV    = dom.Namespace{"env", "http://www.w3.org/2003/05/soap-envelope"}
	NS_ADDRESSING  = dom.Namespace{"a", "http://schemas.xmlsoap.org/ws/2004/08/addressing"}
	NS_CIMBINDING  = dom.Namespace{"b", "http://schemas.dmtf.org/wbem/wsman/1/cimbinding.xsd"}
	NS_ENUM        = dom.Namespace{"n", "http://schemas.xmlsoap.org/ws/2004/09/enumeration"}
	NS_TRANSFER    = dom.Namespace{"x", "http://schemas.xmlsoap.org/ws/2004/09/transfer"}
	NS_WSMAN_DMTF  = dom.Namespace{"w", "http://schemas.dmtf.org/wbem/wsman/1/wsman.xsd"}
	NS_WSMAN_MSFT  = dom.Namespace{"p", "http://schemas.microsoft.com/wbem/wsman/1/wsman.xsd"}
	NS_SCHEMA_INST = dom.Namespace{"xsi", "http://www.w3.org/2001/XMLSchema-instance"}
	NS_WIN_SHELL   = dom.Namespace{"rsp", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell"}
	NS_WSMAN_FAULT = dom.Namespace{"f", "http://schemas.microsoft.com/wbem/wsman/1/wsmanfault"}
)

var MostUsed = [...]dom.Namespace{NS_SOAP_ENV, NS_ADDRESSING, NS_WIN_SHELL, NS_WSMAN_DMTF, NS_WSMAN_MSFT}

func AddUsualNamespaces(node *dom.Element) {
	for _, ns := range MostUsed {
		node.DeclareNamespace(ns)
	}
}

func GetAllNamespaces() []xmlpath.Namespace {
	var ns = []dom.Namespace{NS_WIN_SHELL, NS_ADDRESSING, NS_WSMAN_DMTF, NS_WSMAN_MSFT, NS_SOAP_ENV}

	var xmlpathNs = make([]xmlpath.Namespace, 0, 4)
	for _, namespace := range ns {
		xmlpathNs = append(xmlpathNs, xmlpath.Namespace{Prefix: namespace.Prefix, Uri: namespace.Uri})
	}
	return xmlpathNs
}
