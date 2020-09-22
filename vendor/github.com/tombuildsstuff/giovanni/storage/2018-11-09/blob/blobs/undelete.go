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

// Undelete restores the contents and metadata of soft deleted blob and any associated soft deleted snapshots.
func (client Client) Undelete(ctx context.Context, accountName, containerName, blobName string) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "Undelete", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "Undelete", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "Undelete", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "Undelete", "`blobName` cannot be an empty string.")
	}

	req, err := client.UndeletePreparer(ctx, accountName, containerName, blobName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Undelete", nil, "Failure preparing request")
		return
	}

	resp, err := client.UndeleteSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "Undelete", resp, "Failure sending request")
		return
	}

	result, err = client.UndeleteResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Undelete", resp, "Failure responding to request")
		return
	}

	return
}

// UndeletePreparer prepares the Undelete request.
func (client Client) UndeletePreparer(ctx context.Context, accountName, containerName, blobName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("path", "undelete"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// UndeleteSender sends the Undelete request. The method will close the
// http.Response Body if it receives an error.
func (client Client) UndeleteSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// UndeleteResponder handles the response to the Undelete request. The method always
// closes the http.Response Body.
func (client Client) UndeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
