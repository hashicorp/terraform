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

type AbortCopyInput struct {
	// The Copy ID which should be aborted
	CopyID string

	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string
}

// AbortCopy aborts a pending Copy Blob operation, and leaves a destination blob with zero length and full metadata.
func (client Client) AbortCopy(ctx context.Context, accountName, containerName, blobName string, input AbortCopyInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "AbortCopy", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "AbortCopy", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "AbortCopy", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "AbortCopy", "`blobName` cannot be an empty string.")
	}
	if input.CopyID == "" {
		return result, validation.NewError("blobs.Client", "AbortCopy", "`input.CopyID` cannot be an empty string.")
	}

	req, err := client.AbortCopyPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "AbortCopy", nil, "Failure preparing request")
		return
	}

	resp, err := client.AbortCopySender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "AbortCopy", resp, "Failure sending request")
		return
	}

	result, err = client.AbortCopyResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "AbortCopy", resp, "Failure responding to request")
		return
	}

	return
}

// AbortCopyPreparer prepares the AbortCopy request.
func (client Client) AbortCopyPreparer(ctx context.Context, accountName, containerName, blobName string, input AbortCopyInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp":   autorest.Encode("query", "copy"),
		"copyid": autorest.Encode("query", input.CopyID),
	}

	headers := map[string]interface{}{
		"x-ms-version":     APIVersion,
		"x-ms-copy-action": "abort",
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// AbortCopySender sends the AbortCopy request. The method will close the
// http.Response Body if it receives an error.
func (client Client) AbortCopySender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// AbortCopyResponder handles the response to the AbortCopy request. The method always
// closes the http.Response Body.
func (client Client) AbortCopyResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusNoContent),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
