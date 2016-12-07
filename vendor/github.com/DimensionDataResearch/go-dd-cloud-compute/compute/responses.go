package compute

// APIResponse represents the response to an API call.
type APIResponse interface {
	// GetMessage gets the message associated with the API response.
	GetMessage() string

	// GetResponseCode gets the response code associated with the API response.
	GetResponseCode() string

	// GetAPIVersion gets the version of the API that returned the response.
	GetAPIVersion() string
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

var _ error = &APIError{}

// Well-known API (v1) results

const (
	// ResultSuccess indicates that an operation completed successfully.
	ResultSuccess = "SUCCESS"
)

// Well-known API (v2) response codes

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
