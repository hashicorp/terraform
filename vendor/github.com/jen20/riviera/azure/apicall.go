package azure

// APICall must be implemented by structures which represent requests to the
// ARM API in order that the generic request handling layer has sufficient
// information to execute requests.
type APICall interface {
	APIInfo() APIInfo
}

// APIInfo contains information about a request to the ARM API - which API
// version is required, the HTTP method to use, and a factory function for
// responses.
type APIInfo struct {
	APIVersion            string
	Method                string
	URLPathFunc           func() string
	ResponseTypeFunc      func() interface{}
	RequestPropertiesFunc func() interface{}
	HasBodyOverride       bool
}

// HasBody returns true if the API Request should have a body. This is usually
// the case for PUT, PATCH or POST operations, but is not the case for GET operations.
// TODO(jen20): This may need revisiting at some point.
func (apiInfo APIInfo) HasBody() bool {
	if apiInfo.HasBodyOverride == false {
		return apiInfo.Method == "POST" || apiInfo.Method == "PUT" || apiInfo.Method == "PATCH"
	}
	return !(apiInfo.Method == "POST" || apiInfo.Method == "PUT" || apiInfo.Method == "PATCH")
}
