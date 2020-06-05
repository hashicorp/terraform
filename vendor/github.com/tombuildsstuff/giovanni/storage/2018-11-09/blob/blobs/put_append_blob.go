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

type PutAppendBlobInput struct {
	CacheControl       *string
	ContentDisposition *string
	ContentEncoding    *string
	ContentLanguage    *string
	ContentMD5         *string
	ContentType        *string
	LeaseID            *string
	MetaData           map[string]string
}

// PutAppendBlob is a wrapper around the Put API call (with a stricter input object)
// which creates a new append blob, or updates the content of an existing blob.
func (client Client) PutAppendBlob(ctx context.Context, accountName, containerName, blobName string, input PutAppendBlobInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutAppendBlob", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutAppendBlob", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutAppendBlob", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutAppendBlob", "`blobName` cannot be an empty string.")
	}
	if err := metadata.Validate(input.MetaData); err != nil {
		return result, validation.NewError("blobs.Client", "PutAppendBlob", fmt.Sprintf("`input.MetaData` is not valid: %s.", err))
	}

	req, err := client.PutAppendBlobPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutAppendBlob", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutAppendBlobSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutAppendBlob", resp, "Failure sending request")
		return
	}

	result, err = client.PutAppendBlobResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutAppendBlob", resp, "Failure responding to request")
		return
	}

	return
}

// PutAppendBlobPreparer prepares the PutAppendBlob request.
func (client Client) PutAppendBlobPreparer(ctx context.Context, accountName, containerName, blobName string, input PutAppendBlobInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-blob-type": string(AppendBlob),
		"x-ms-version":   APIVersion,

		// For a page blob or an append blob, the value of this header must be set to zero,
		// as Put Blob is used only to initialize the blob
		"Content-Length": 0,
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

	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutAppendBlobSender sends the PutAppendBlob request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutAppendBlobSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutAppendBlobResponder handles the response to the PutAppendBlob request. The method always
// closes the http.Response Body.
func (client Client) PutAppendBlobResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
