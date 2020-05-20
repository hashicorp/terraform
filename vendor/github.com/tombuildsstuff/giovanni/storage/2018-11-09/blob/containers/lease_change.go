package containers

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type ChangeLeaseInput struct {
	ExistingLeaseID string
	ProposedLeaseID string
}

type ChangeLeaseResponse struct {
	autorest.Response

	LeaseID string
}

// ChangeLease changes the lock from one Lease ID to another Lease ID
func (client Client) ChangeLease(ctx context.Context, accountName, containerName string, input ChangeLeaseInput) (result ChangeLeaseResponse, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "ChangeLease", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "ChangeLease", "`containerName` cannot be an empty string.")
	}
	if input.ExistingLeaseID == "" {
		return result, validation.NewError("containers.Client", "ChangeLease", "`input.ExistingLeaseID` cannot be an empty string.")
	}
	if input.ProposedLeaseID == "" {
		return result, validation.NewError("containers.Client", "ChangeLease", "`input.ProposedLeaseID` cannot be an empty string.")
	}

	req, err := client.ChangeLeasePreparer(ctx, accountName, containerName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "ChangeLease", nil, "Failure preparing request")
		return
	}

	resp, err := client.ChangeLeaseSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "ChangeLease", resp, "Failure sending request")
		return
	}

	result, err = client.ChangeLeaseResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "ChangeLease", resp, "Failure responding to request")
		return
	}

	return
}

// ChangeLeasePreparer prepares the ChangeLease request.
func (client Client) ChangeLeasePreparer(ctx context.Context, accountName string, containerName string, input ChangeLeaseInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"restype": autorest.Encode("path", "container"),
		"comp":    autorest.Encode("path", "lease"),
	}

	headers := map[string]interface{}{
		"x-ms-version":           APIVersion,
		"x-ms-lease-action":      "change",
		"x-ms-lease-id":          input.ExistingLeaseID,
		"x-ms-proposed-lease-id": input.ProposedLeaseID,
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

// ChangeLeaseSender sends the ChangeLease request. The method will close the
// http.Response Body if it receives an error.
func (client Client) ChangeLeaseSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ChangeLeaseResponder handles the response to the ChangeLease request. The method always
// closes the http.Response Body.
func (client Client) ChangeLeaseResponder(resp *http.Response) (result ChangeLeaseResponse, err error) {
	if resp != nil {
		result.LeaseID = resp.Header.Get("x-ms-lease-id")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
