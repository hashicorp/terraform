package compute

import "fmt"

// APIResponse represents the basic response most commonly received when making API calls.
type APIResponse struct {
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

// ToError creates an error representing the API response.
func (response *APIResponse) ToError(errorMessageOrFormat string, formatArgs ...interface{}) error {
	return &APIError{
		Message:  fmt.Sprintf(errorMessageOrFormat, formatArgs...),
		Response: *response,
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
	return apiError.Message
}

// ResponseCode returns the response code associated with the APIError.
func (apiError *APIError) ResponseCode() string {
	return apiError.Response.ResponseCode
}

var _ error = &APIError{}

// Well-known API response codes

const (
	// ResponseCodeOK indicates that an operation completed successfully.
	ResponseCodeOK = "OK"

	// ResponseCodeInProgress indicates that an operation is in progress.
	ResponseCodeInProgress = "IN_PROGRESS"

	// ResponseCodeResourceNotFound indicates that an operation failed because a target resource was not found.
	ResponseCodeResourceNotFound = "RESOURCE_NOT_FOUND"

	// ResponseCodeAuthorizationFailure indicates that an operation failed because the caller was not authorised to perform that operation (e.g. target resource belongs to another organisation).
	ResponseCodeAuthorizationFailure = "AUTHORIZATION_FAILURE"

	// ResponseCodeInvalidInputData indicates that an operation failed due to invalid input data.
	ResponseCodeInvalidInputData = "INVALID_INPUT_DATA"

	// ResponseCodeResourceNameNotUnique indicates that an operation failed due to the use of a name that duplicates an existing name.
	ResponseCodeResourceNameNotUnique = "NAME_NOT_UNIQUE"

	// ResponseCodeIPAddressNotUnique indicates that an operation failed due to the use of an IP address that duplicates an existing IP address.
	ResponseCodeIPAddressNotUnique = "IP_ADDRESS_NOT_UNIQUE"

	// ResponseCodeIPAddressOutOfRange indicates that an operation failed due to the use of an IP address lies outside the supported range (e.g. outside of the target subnet).
	ResponseCodeIPAddressOutOfRange = "IP_ADDRESS_OUT_OF_RANGE"

	// ResponseCodeNoIPAddressAvailable indicates that there are no remaining unreserved IPv4 addresses in the target subnet.
	ResponseCodeNoIPAddressAvailable = "NO_IP_ADDRESS_AVAILABLE"

	// ResponseCodeResourceHasDependency indicates that an operation cannot be performed on a resource because of a resource that depends on it.
	ResponseCodeResourceHasDependency = "HAS_DEPENDENCY"

	// ResponseCodeResourceBusy indicates that an operation cannot be performed on a resource because the resource is busy.
	ResponseCodeResourceBusy = "RESOURCE_BUSY"

	// ResponseCodeResourceLocked indicates that an operation cannot be performed on a resource because the resource is locked.
	ResponseCodeResourceLocked = "RESOURCE_LOCKED"

	// ResponseCodeExceedsLimit indicates that an operation failed because a resource limit was exceeded.
	ResponseCodeExceedsLimit = "EXCEEDS_LIMIT"

	// ResponseCodeOutOfResources indicates that an operation failed because some type of resource (e.g. free IPv4 addresses) has been exhausted.
	ResponseCodeOutOfResources = "OUT_OF_RESOURCES"

	// ResponseCodeOperationNotSupported indicates that an operation is not supported.
	ResponseCodeOperationNotSupported = "OPERATION_NOT_SUPPORTED"

	// ResponseCodeInfrastructureInMaintenance indicates that an operation failed due to maintenance being performed on the supporting infrastructure.
	ResponseCodeInfrastructureInMaintenance = "INFRASTRUCTURE_IN_MAINTENANCE"
)
