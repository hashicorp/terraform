package service

import "encoding/xml"

// ApplicationsList top level element.
type ApplicationsList struct {
	XMLName      xml.Name             `xml:"list"`
	Applications []ApplicationService `xml:"application"`
}

// ApplicationService - object within ApplicationsList.
type ApplicationService struct {
	XMLName     xml.Name  `xml:"application"`
	Name        string    `xml:"name"`
	ObjectID    string    `xml:"objectId,omitempty"`
	Type        string    `xml:"type,omitempty>TypeName,omitempty"`
	Revision    int       `xml:"revision,omitempty"`
	Description string    `xml:"description"`
	Element     []Element `xml:"element"`
}

// Element - object within ApplicationService
type Element struct {
	XMLName             xml.Name `xml:"element"`
	ApplicationProtocol string   `xml:"applicationProtocol"`
	Value               string   `xml:"value"`
}
