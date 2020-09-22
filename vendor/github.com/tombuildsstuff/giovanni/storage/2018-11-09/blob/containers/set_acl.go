package containers

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

// SetAccessControl sets the Access Control for a Container without a Lease ID
func (client Client) SetAccessControl(ctx context.Context, accountName, containerName string, level AccessLevel) (autorest.Response, error) {
	return client.SetAccessControlWithLeaseID(ctx, accountName, containerName, "", level)
}

// SetAccessControlWithLeaseID sets the Access Control for a Container using the specified Lease ID
func (client Client) SetAccessControlWithLeaseID(ctx context.Context, accountName, containerName, leaseID string, level AccessLevel) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "SetAccessControl", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "SetAccessControl", "`containerName` cannot be an empty string.")
	}

	req, err := client.SetAccessControlWithLeaseIDPreparer(ctx, accountName, containerName, leaseID, level)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "SetAccessControl", nil, "Failure preparing request")
		return
	}

	resp, err := client.SetAccessControlWithLeaseIDSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "SetAccessControl", resp, "Failure sending request")
		return
	}

	result, err = client.SetAccessControlWithLeaseIDResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "SetAccessControl", resp, "Failure responding to request")
		return
	}

	return
}

// SetAccessControlWithLeaseIDPreparer prepares the SetAccessControlWithLeaseID request.
func (client Client) SetAccessControlWithLeaseIDPreparer(ctx context.Context, accountName, containerName, leaseID string, level AccessLevel) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"comp":    autorest.Encode("path", "acl"),
		"restype": autorest.Encode("path", "container"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	headers = client.setAccessLevelIntoHeaders(headers, level)

	// If specified, Get Container Properties only succeeds if the container’s lease is active and matches this ID.
	// If there is no active lease or the ID does not match, 412 (Precondition Failed) is returned.
	if leaseID != "" {
		headers["x-ms-lease-id"] = leaseID
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

// SetAccessControlWithLeaseIDSender sends the SetAccessControlWithLeaseID request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SetAccessControlWithLeaseIDSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SetAccessControlWithLeaseIDResponder handles the response to the SetAccessControlWithLeaseID request. The method always
// closes the http.Response Body.
func (client Client) SetAccessControlWithLeaseIDResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
