package containers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

// SetMetaData sets the specified MetaData on the Container without a Lease ID
func (client Client) SetMetaData(ctx context.Context, accountName, containerName string, metaData map[string]string) (autorest.Response, error) {
	return client.SetMetaDataWithLeaseID(ctx, accountName, containerName, "", metaData)
}

// SetMetaDataWithLeaseID sets the specified MetaData on the Container using the specified Lease ID
func (client Client) SetMetaDataWithLeaseID(ctx context.Context, accountName, containerName, leaseID string, metaData map[string]string) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "SetMetaData", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "SetMetaData", "`containerName` cannot be an empty string.")
	}
	if err := metadata.Validate(metaData); err != nil {
		return result, validation.NewError("containers.Client", "SetMetaData", fmt.Sprintf("`metaData` is not valid: %s.", err))
	}

	req, err := client.SetMetaDataWithLeaseIDPreparer(ctx, accountName, containerName, leaseID, metaData)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "SetMetaData", nil, "Failure preparing request")
		return
	}

	resp, err := client.SetMetaDataWithLeaseIDSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "SetMetaData", resp, "Failure sending request")
		return
	}

	result, err = client.SetMetaDataWithLeaseIDResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "SetMetaData", resp, "Failure responding to request")
		return
	}

	return
}

// SetMetaDataWithLeaseIDPreparer prepares the SetMetaDataWithLeaseID request.
func (client Client) SetMetaDataWithLeaseIDPreparer(ctx context.Context, accountName, containerName, leaseID string, metaData map[string]string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"comp":    autorest.Encode("path", "metadata"),
		"restype": autorest.Encode("path", "container"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	headers = metadata.SetIntoHeaders(headers, metaData)

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

// SetMetaDataWithLeaseIDSender sends the SetMetaDataWithLeaseID request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SetMetaDataWithLeaseIDSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SetMetaDataWithLeaseIDResponder handles the response to the SetMetaDataWithLeaseID request. The method always
// closes the http.Response Body.
func (client Client) SetMetaDataWithLeaseIDResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
