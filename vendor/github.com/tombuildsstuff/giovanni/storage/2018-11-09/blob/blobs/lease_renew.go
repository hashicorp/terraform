package blobs

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

func (client Client) RenewLease(ctx context.Context, accountName, containerName, blobName, leaseID string) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "RenewLease", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "RenewLease", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "RenewLease", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "RenewLease", "`blobName` cannot be an empty string.")
	}
	if leaseID == "" {
		return result, validation.NewError("blobs.Client", "RenewLease", "`leaseID` cannot be an empty string.")
	}

	req, err := client.RenewLeasePreparer(ctx, accountName, containerName, blobName, leaseID)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "RenewLease", nil, "Failure preparing request")
		return
	}

	resp, err := client.RenewLeaseSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "RenewLease", resp, "Failure sending request")
		return
	}

	result, err = client.RenewLeaseResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "RenewLease", resp, "Failure responding to request")
		return
	}

	return
}

// RenewLeasePreparer prepares the RenewLease request.
func (client Client) RenewLeasePreparer(ctx context.Context, accountName, containerName, blobName, leaseID string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "lease"),
	}

	headers := map[string]interface{}{
		"x-ms-version":      APIVersion,
		"x-ms-lease-action": "renew",
		"x-ms-lease-id":     leaseID,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// RenewLeaseSender sends the RenewLease request. The method will close the
// http.Response Body if it receives an error.
func (client Client) RenewLeaseSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// RenewLeaseResponder handles the response to the RenewLease request. The method always
// closes the http.Response Body.
func (client Client) RenewLeaseResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
