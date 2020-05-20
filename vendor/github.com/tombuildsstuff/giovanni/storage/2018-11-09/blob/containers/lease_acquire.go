package containers

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type AcquireLeaseInput struct {
	// Specifies the duration of the lease, in seconds, or negative one (-1) for a lease that never expires.
	// A non-infinite lease can be between 15 and 60 seconds
	LeaseDuration int

	ProposedLeaseID string
}

type AcquireLeaseResponse struct {
	autorest.Response

	LeaseID string
}

// AcquireLease establishes and manages a lock on a container for delete operations.
func (client Client) AcquireLease(ctx context.Context, accountName, containerName string, input AcquireLeaseInput) (result AcquireLeaseResponse, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "AcquireLease", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "AcquireLease", "`containerName` cannot be an empty string.")
	}
	// An infinite lease duration is -1 seconds. A non-infinite lease can be between 15 and 60 seconds
	if input.LeaseDuration != -1 && (input.LeaseDuration <= 15 || input.LeaseDuration >= 60) {
		return result, validation.NewError("containers.Client", "AcquireLease", "`input.LeaseDuration` must be -1 (infinite), or between 15 and 60 seconds.")
	}

	req, err := client.AcquireLeasePreparer(ctx, accountName, containerName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "AcquireLease", nil, "Failure preparing request")
		return
	}

	resp, err := client.AcquireLeaseSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "AcquireLease", resp, "Failure sending request")
		return
	}

	result, err = client.AcquireLeaseResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "AcquireLease", resp, "Failure responding to request")
		return
	}

	return
}

// AcquireLeasePreparer prepares the AcquireLease request.
func (client Client) AcquireLeasePreparer(ctx context.Context, accountName string, containerName string, input AcquireLeaseInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"restype": autorest.Encode("path", "container"),
		"comp":    autorest.Encode("path", "lease"),
	}

	headers := map[string]interface{}{
		"x-ms-version":        APIVersion,
		"x-ms-lease-action":   "acquire",
		"x-ms-lease-duration": input.LeaseDuration,
	}

	if input.ProposedLeaseID != "" {
		headers["x-ms-proposed-lease-id"] = input.ProposedLeaseID
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

// AcquireLeaseSender sends the AcquireLease request. The method will close the
// http.Response Body if it receives an error.
func (client Client) AcquireLeaseSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// AcquireLeaseResponder handles the response to the AcquireLease request. The method always
// closes the http.Response Body.
func (client Client) AcquireLeaseResponder(resp *http.Response) (result AcquireLeaseResponse, err error) {
	if resp != nil {
		result.LeaseID = resp.Header.Get("x-ms-lease-id")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
