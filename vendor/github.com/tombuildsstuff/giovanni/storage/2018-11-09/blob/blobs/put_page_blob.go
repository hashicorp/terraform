package blobs

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

type PutPageBlobInput struct {
	CacheControl       *string
	ContentDisposition *string
	ContentEncoding    *string
	ContentLanguage    *string
	ContentMD5         *string
	ContentType        *string
	LeaseID            *string
	MetaData           map[string]string

	BlobContentLengthBytes int64
	BlobSequenceNumber     *int64
	AccessTier             *AccessTier
}

// PutPageBlob is a wrapper around the Put API call (with a stricter input object)
// which creates a new block blob, or updates the content of an existing page blob.
func (client Client) PutPageBlob(ctx context.Context, accountName, containerName, blobName string, input PutPageBlobInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutPageBlob", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutPageBlob", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutPageBlob", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutPageBlob", "`blobName` cannot be an empty string.")
	}
	if input.BlobContentLengthBytes == 0 || input.BlobContentLengthBytes%512 != 0 {
		return result, validation.NewError("blobs.Client", "PutPageBlob", "`input.BlobContentLengthBytes` must be aligned to a 512-byte boundary.")
	}

	req, err := client.PutPageBlobPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageBlob", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutPageBlobSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageBlob", resp, "Failure sending request")
		return
	}

	result, err = client.PutPageBlobResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageBlob", resp, "Failure responding to request")
		return
	}

	return
}

// PutPageBlobPreparer prepares the PutPageBlob request.
func (client Client) PutPageBlobPreparer(ctx context.Context, accountName, containerName, blobName string, input PutPageBlobInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-blob-type": string(PageBlob),
		"x-ms-version":   APIVersion,

		// For a page blob or an page blob, the value of this header must be set to zero,
		// as Put Blob is used only to initialize the blob
		"Content-Length": 0,

		// This header specifies the maximum size for the page blob, up to 8 TB.
		// The page blob size must be aligned to a 512-byte boundary.
		"x-ms-blob-content-length": input.BlobContentLengthBytes,
	}

	if input.AccessTier != nil {
		headers["x-ms-access-tier"] = string(*input.AccessTier)
	}
	if input.BlobSequenceNumber != nil {
		headers["x-ms-blob-sequence-number"] = *input.BlobSequenceNumber
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

// PutPageBlobSender sends the PutPageBlob request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutPageBlobSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutPageBlobResponder handles the response to the PutPageBlob request. The method always
// closes the http.Response Body.
func (client Client) PutPageBlobResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
