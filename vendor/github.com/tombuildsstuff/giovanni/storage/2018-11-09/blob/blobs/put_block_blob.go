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

type PutBlockBlobInput struct {
	CacheControl       *string
	Content            *[]byte
	ContentDisposition *string
	ContentEncoding    *string
	ContentLanguage    *string
	ContentMD5         *string
	ContentType        *string
	LeaseID            *string
	MetaData           map[string]string
}

// PutBlockBlob is a wrapper around the Put API call (with a stricter input object)
// which creates a new block append blob, or updates the content of an existing block blob.
func (client Client) PutBlockBlob(ctx context.Context, accountName, containerName, blobName string, input PutBlockBlobInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", "`blobName` cannot be an empty string.")
	}
	if input.Content != nil && len(*input.Content) == 0 {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", "`input.Content` must either be nil or not empty.")
	}
	if err := metadata.Validate(input.MetaData); err != nil {
		return result, validation.NewError("blobs.Client", "PutBlockBlob", fmt.Sprintf("`input.MetaData` is not valid: %s.", err))
	}

	req, err := client.PutBlockBlobPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockBlob", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutBlockBlobSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockBlob", resp, "Failure sending request")
		return
	}

	result, err = client.PutBlockBlobResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockBlob", resp, "Failure responding to request")
		return
	}

	return
}

// PutBlockBlobPreparer prepares the PutBlockBlob request.
func (client Client) PutBlockBlobPreparer(ctx context.Context, accountName, containerName, blobName string, input PutBlockBlobInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-blob-type": string(BlockBlob),
		"x-ms-version":   APIVersion,
	}

	if input.CacheControl != nil {
		headers["x-ms-blob-cache-control"] = *input.CacheControl
	}
	if input.ContentDisposition != nil {
		headers["x-ms-blob-content-disposition"] = *input.ContentDisposition
	}
	if input.ContentEncoding != nil {
		headers["x-ms-blob-content-encoding"] = *input.ContentEncoding
	}
	if input.ContentLanguage != nil {
		headers["x-ms-blob-content-language"] = *input.ContentLanguage
	}
	if input.ContentMD5 != nil {
		headers["x-ms-blob-content-md5"] = *input.ContentMD5
	}
	if input.ContentType != nil {
		headers["x-ms-blob-content-type"] = *input.ContentType
	}
	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.Content != nil {
		headers["Content-Length"] = int(len(*input.Content))
	}

	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	decorators := []autorest.PrepareDecorator{
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
	}

	if input.Content != nil {
		decorators = append(decorators, autorest.WithBytes(input.Content))
	}

	preparer := autorest.CreatePreparer(decorators...)
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutBlockBlobSender sends the PutBlockBlob request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutBlockBlobSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutBlockBlobResponder handles the response to the PutBlockBlob request. The method always
// closes the http.Response Body.
func (client Client) PutBlockBlobResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
