package rundeck

import "encoding/xml"

// Error implements the error interface for a Rundeck API error that was
// returned from the server as XML.
type Error struct {
	XMLName xml.Name `xml:"result"`
	IsError bool     `xml:"error,attr"`
	Message string   `xml:"error>message"`
}

func (err Error) Error() string {
	return err.Message
}

type NotFoundError struct{}

func (err NotFoundError) Error() string {
	return "not found"
}
