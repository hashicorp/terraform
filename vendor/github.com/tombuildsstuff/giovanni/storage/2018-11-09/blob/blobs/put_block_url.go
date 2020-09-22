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

type PutBlockFromURLInput struct {
	BlockID    string
	CopySource string

	ContentMD5 *string
	LeaseID    *string
	Range      *string
}

type PutBlockFromURLResult struct {
	autorest.Response
	ContentMD5 string
}

// PutBlockFromURL creates a new block to be committed as part of a blob where the contents are read from a URL
func (client Client) PutBlockFromURL(ctx context.Context, accountName, containerName, blobName string, input PutBlockFromURLInput) (result PutBlockFromURLResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`blobName` cannot be an empty string.")
	}
	if input.BlockID == "" {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`input.BlockID` cannot be an empty string.")
	}
	if input.CopySource == "" {
		return result, validation.NewError("blobs.Client", "PutBlockFromURL", "`input.CopySource` cannot be an empty string.")
	}

	req, err := client.PutBlockFromURLPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockFromURL", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutBlockFromURLSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockFromURL", resp, "Failure sending request")
		return
	}

	result, err = client.PutBlockFromURLResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockFromURL", resp, "Failure responding to request")
		return
	}

	return
}

// PutBlockFromURLPreparer prepares the PutBlockFromURL request.
func (client Client) PutBlockFromURLPreparer(ctx context.Context, accountName, containerName, blobName string, input PutBlockFromURLInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp":    autorest.Encode("query", "block"),
		"blockid": autorest.Encode("query", input.BlockID),
	}

	headers := map[string]interface{}{
		"x-ms-version":     APIVersion,
		"x-ms-copy-source": input.CopySource,
	}

	if input.ContentMD5 != nil {
		headers["x-ms-source-content-md5"] = *input.ContentMD5
	}
	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.Range != nil {
		headers["x-ms-source-range"] = *input.Range
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutBlockFromURLSender sends the PutBlockFromURL request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutBlockFromURLSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutBlockFromURLResponder handles the response to the PutBlockFromURL request. The method always
// closes the http.Response Body.
func (client Client) PutBlockFromURLResponder(resp *http.Response) (result PutBlockFromURLResult, err error) {
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
