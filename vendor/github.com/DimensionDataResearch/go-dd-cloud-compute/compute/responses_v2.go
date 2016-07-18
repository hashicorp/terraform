package compute

import "fmt"

// APIResponseV2 represents the basic response most commonly received when making v2 API calls.
type APIResponseV2 struct {
	// The operation that was performed.
	Operation string `json:"operation"`

	// The API response code.
	ResponseCode string `json:"responseCode"`

	// The API status message (if any).
	Message string `json:"message"`

	// Informational messages (if any) relating to request fields.
	FieldMessages []FieldMessage `json:"info"`

	// Warning messages (if any) relating to request fields.
	FieldWarnings []FieldMessage `json:"warning"`

	// Error messages (if any) relating to request fields.
	FieldErrors []FieldMessage `json:"error"`

	// The request ID (correlation identifier).
	RequestID string `json:"requestId"`
}

// GetMessage gets the message associated with the API response.
func (response *APIResponseV2) GetMessage() string {
	return response.Message
}

// GetResponseCode gets the response code associated with the API response.
func (response *APIResponseV2) GetResponseCode() string {
	return response.ResponseCode
}

// GetAPIVersion gets the response code associated with the API response.
func (response *APIResponseV2) GetAPIVersion() string {
	return "v2"
}

var _ APIResponse = &APIResponseV2{}

// ToError creates an error representing the API response.
func (response *APIResponseV2) ToError(errorMessageOrFormat string, formatArgs ...interface{}) error {
	return &APIError{
		Message:  fmt.Sprintf(errorMessageOrFormat, formatArgs...),
		Response: response,
	}
}

// FieldMessage represents a field name together with an associated message.
type FieldMessage struct {
	// The field name.
	FieldName string `json:"name"`

	// The field message.
	Message string `json:"value"`
}

// APIError is an error representing an error response from an API.
type APIError struct {
	Message  string
	Response APIResponse
}

// Error returns the error message associated with the APIError.
func (apiError *APIError) Error() string {
	return apiError.Error()
}

var _ error = &APIError{}
