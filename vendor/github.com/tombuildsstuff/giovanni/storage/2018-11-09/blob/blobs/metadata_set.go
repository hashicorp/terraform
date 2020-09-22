package blobs

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

type SetMetaDataInput struct {
	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string

	// Any metadata which should be added to this blob
	MetaData map[string]string
}

// SetMetaData marks the specified blob or snapshot for deletion. The blob is later deleted during garbage collection.
func (client Client) SetMetaData(ctx context.Context, accountName, containerName, blobName string, input SetMetaDataInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "GetProperties", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "GetProperties", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "GetProperties", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "GetProperties", "`blobName` cannot be an empty string.")
	}
	if err := metadata.Validate(input.MetaData); err != nil {
		return result, validation.NewError("blobs.Client", "GetProperties", fmt.Sprintf("`input.MetaData` is not valid: %s.", err))
	}

	req, err := client.SetMetaDataPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetMetaData", nil, "Failure preparing request")
		return
	}

	resp, err := client.SetMetaDataSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetMetaData", resp, "Failure sending request")
		return
	}

	result, err = client.SetMetaDataResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetMetaData", resp, "Failure responding to request")
		return
	}

	return
}

// SetMetaDataPreparer prepares the SetMetaData request.
func (client Client) SetMetaDataPreparer(ctx context.Context, accountName, containerName, blobName string, input SetMetaDataInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "metadata"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// SetMetaDataSender sends the SetMetaData request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SetMetaDataSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SetMetaDataResponder handles the response to the SetMetaData request. The method always
// closes the http.Response Body.
func (client Client) SetMetaDataResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
