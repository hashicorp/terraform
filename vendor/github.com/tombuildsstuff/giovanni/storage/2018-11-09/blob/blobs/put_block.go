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

type PutBlockInput struct {
	BlockID    string
	Content    []byte
	ContentMD5 *string
	LeaseID    *string
}

type PutBlockResult struct {
	autorest.Response

	ContentMD5 string
}

// PutBlock creates a new block to be committed as part of a blob.
func (client Client) PutBlock(ctx context.Context, accountName, containerName, blobName string, input PutBlockInput) (result PutBlockResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutBlock", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutBlock", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutBlock", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutBlock", "`blobName` cannot be an empty string.")
	}
	if input.BlockID == "" {
		return result, validation.NewError("blobs.Client", "PutBlock", "`input.BlockID` cannot be an empty string.")
	}
	if len(input.Content) == 0 {
		return result, validation.NewError("blobs.Client", "PutBlock", "`input.Content` cannot be empty.")
	}

	req, err := client.PutBlockPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlock", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutBlockSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlock", resp, "Failure sending request")
		return
	}

	result, err = client.PutBlockResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlock", resp, "Failure responding to request")
		return
	}

	return
}

// PutBlockPreparer prepares the PutBlock request.
func (client Client) PutBlockPreparer(ctx context.Context, accountName, containerName, blobName string, input PutBlockInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp":    autorest.Encode("query", "block"),
		"blockid": autorest.Encode("query", input.BlockID),
	}

	headers := map[string]interface{}{
		"x-ms-version":   APIVersion,
		"Content-Length": int(len(input.Content)),
	}

	if input.ContentMD5 != nil {
		headers["x-ms-blob-content-md5"] = *input.ContentMD5
	}
	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers),
		autorest.WithBytes(&input.Content))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutBlockSender sends the PutBlock request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutBlockSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutBlockResponder handles the response to the PutBlock request. The method always
// closes the http.Response Body.
func (client Client) PutBlockResponder(resp *http.Response) (result PutBlockResult, err error) {
	if resp != nil && resp.Header != nil {
		result.ContentMD5 = resp.Header.Get("Content-MD5")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
