/*
Package azure provides Azure-specific implementations used with AutoRest.

See the included examples for more detail.
*/
package azure

import (
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/github.com/Azure/go-autorest/autorest"
)

const (
	// HeaderClientID is the Azure extension header to set a user-specified request ID.
	HeaderClientID = "x-ms-client-request-id"

	// HeaderReturnClientID is the Azure extension header to set if the user-specified request ID
	// should be included in the response.
	HeaderReturnClientID = "x-ms-return-client-request-id"

	// HeaderRequestID is the Azure extension header of the service generated request ID returned
	// in the response.
	HeaderRequestID = "x-ms-request-id"
)

// WithReturningClientID returns a PrepareDecorator that adds an HTTP extension header of
// x-ms-client-request-id whose value is the passed, undecorated UUID (e.g.,
// "0F39878C-5F76-4DB8-A25D-61D2C193C3CA"). It also sets the x-ms-return-client-request-id
// header to true such that UUID accompanies the http.Response.
func WithReturningClientID(uuid string) autorest.PrepareDecorator {
	preparer := autorest.CreatePreparer(
		WithClientID(uuid),
		WithReturnClientID(true))

	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}
			return preparer.Prepare(r)
		})
	}
}

// WithClientID returns a PrepareDecorator that adds an HTTP extension header of
// x-ms-client-request-id whose value is passed, undecorated UUID (e.g.,
// "0F39878C-5F76-4DB8-A25D-61D2C193C3CA").
func WithClientID(uuid string) autorest.PrepareDecorator {
	return autorest.WithHeader(HeaderClientID, uuid)
}

// WithReturnClientID returns a PrepareDecorator that adds an HTTP extension header of
// x-ms-return-client-request-id whose boolean value indicates if the value of the
// x-ms-client-request-id header should be included in the http.Response.
func WithReturnClientID(b bool) autorest.PrepareDecorator {
	return autorest.WithHeader(HeaderReturnClientID, strconv.FormatBool(b))
}

// ExtractClientID extracts the client identifier from the x-ms-client-request-id header set on the
// http.Request sent to the service (and returned in the http.Response)
func ExtractClientID(resp *http.Response) string {
	return autorest.ExtractHeaderValue(HeaderClientID, resp)
}

// ExtractRequestID extracts the Azure server generated request identifier from the
// x-ms-request-id header.
func ExtractRequestID(resp *http.Response) string {
	return autorest.ExtractHeaderValue(HeaderRequestID, resp)
}
