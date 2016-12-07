package compute

import (
	"encoding/xml"
	"fmt"
)

// APIResponseV1 represents a response from the CloudControl v1 API for an asynchronous operation.
type APIResponseV1 struct {
	// The XML name for the "APIResponseV1" data contract
	XMLName xml.Name `xml:"Status"`

	// The operation for which status is being reported.
	Operation string `xml:"operation"`

	// The operation result.
	Result string `xml:"result"`

	// A brief message describing the operation result.
	Message string `xml:"resultDetail"`

	// The operation result code
	ResultCode string `xml:"resultCode"`
}

// GetMessage gets the message associated with the API response.
func (response *APIResponseV1) GetMessage() string {
	return response.Message
}

// GetResponseCode gets the response code associated with the API response.
func (response *APIResponseV1) GetResponseCode() string {
	return response.Result
}

// GetAPIVersion gets the response code associated with the API response.
func (response *APIResponseV1) GetAPIVersion() string {
	return "v1"
}

var _ APIResponse = &APIResponseV1{}

// ToError creates an error representing the API response.
func (response *APIResponseV1) ToError(errorMessageOrFormat string, formatArgs ...interface{}) error {
	return &APIError{
		Message:  fmt.Sprintf(errorMessageOrFormat, formatArgs...),
		Response: response,
	}
}
