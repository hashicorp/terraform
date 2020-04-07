package containers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type BreakLeaseInput struct {
	//  For a break operation, proposed duration the lease should continue
	//  before it is broken, in seconds, between 0 and 60.
	//  This break period is only used if it is shorter than the time remaining on the lease.
	//  If longer, the time remaining on the lease is used.
	//  A new lease will not be available before the break period has expired,
	//  but the lease may be held for longer than the break period.
	//  If this header does not appear with a break operation, a fixed-duration lease breaks
	//  after the remaining lease period elapses, and an infinite lease breaks immediately.
	BreakPeriod *int

	LeaseID string
}

type BreakLeaseResponse struct {
	autorest.Response

	// Approximate time remaining in the lease period, in seconds.
	// If the break is immediate, 0 is returned.
	LeaseTime int
}

// BreakLease breaks a lock based on it's Lease ID
func (client Client) BreakLease(ctx context.Context, accountName, containerName string, input BreakLeaseInput) (result BreakLeaseResponse, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "BreakLease", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "BreakLease", "`containerName` cannot be an empty string.")
	}
	if input.LeaseID == "" {
		return result, validation.NewError("containers.Client", "BreakLease", "`input.LeaseID` cannot be an empty string.")
	}

	req, err := client.BreakLeasePreparer(ctx, accountName, containerName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "BreakLease", nil, "Failure preparing request")
		return
	}

	resp, err := client.BreakLeaseSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "BreakLease", resp, "Failure sending request")
		return
	}

	result, err = client.BreakLeaseResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "BreakLease", resp, "Failure responding to request")
		return
	}

	return
}

// BreakLeasePreparer prepares the BreakLease request.
func (client Client) BreakLeasePreparer(ctx context.Context, accountName string, containerName string, input BreakLeaseInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"restype": autorest.Encode("path", "container"),
		"comp":    autorest.Encode("path", "lease"),
	}

	headers := map[string]interface{}{
		"x-ms-version":      APIVersion,
		"x-ms-lease-action": "break",
		"x-ms-lease-id":     input.LeaseID,
	}

	if input.BreakPeriod != nil {
		headers["x-ms-lease-break-period"] = *input.BreakPeriod
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/xml; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// BreakLeaseSender sends the BreakLease request. The method will close the
// http.Response Body if it receives an error.
func (client Client) BreakLeaseSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// BreakLeaseResponder handles the response to the BreakLease request. The method always
// closes the http.Response Body.
func (client Client) BreakLeaseResponder(resp *http.Response) (result BreakLeaseResponse, err error) {
	if resp != nil {
		leaseRaw := resp.Header.Get("x-ms-lease-time")
		if leaseRaw != "" {
			i, err := strconv.Atoi(leaseRaw)
			if err == nil {
				result.LeaseTime = i
			}
		}
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
