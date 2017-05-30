package api

// NSXApi object.
type NSXApi interface {
	Method() string
	Endpoint() string
	RequestObject() interface{}
	ResponseObject() interface{}
	StatusCode() int
	RawResponse() []byte
	Error() error

	SetResponseObject(interface{})
	SetStatusCode(int)
	SetRawResponse([]byte)
}
