package blobs

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type AppendBlockInput struct {

	// A number indicating the byte offset to compare.
	// Append Block will succeed only if the append position is equal to this number.
	// If it is not, the request will fail with an AppendPositionConditionNotMet
	// error (HTTP status code 412 – Precondition Failed)
	BlobConditionAppendPosition *int64

	// The max length in bytes permitted for the append blob.
	// If the Append Block operation would cause the blob to exceed that limit or if the blob size
	// is already greater than the value specified in this header, the request will fail with
	// an MaxBlobSizeConditionNotMet error (HTTP status code 412 – Precondition Failed).
	BlobConditionMaxSize *int64

	// The Bytes which should be appended to the end of this Append Blob.
	// This can either be nil, which creates an empty blob, or a byte array
	Content *[]byte

	// An MD5 hash of the block content.
	// This hash is used to verify the integrity of the block during transport.
	// When this header is specified, the storage service compares the hash of the content
	// that has arrived with this header value.
	//
	// Note that this MD5 hash is not stored with the blob.
	// If the two hashes do not match, the operation will fail with error code 400 (Bad Request).
	ContentMD5 *string

	// Required if the blob has an active lease.
	// To perform this operation on a blob with an active lease, specify the valid lease ID for this header.
	LeaseID *string
}

type AppendBlockResult struct {
	autorest.Response

	BlobAppendOffset        string
	BlobCommittedBlockCount int64
	ContentMD5              string
	ETag                    string
	LastModified            string
}

// AppendBlock commits a new block of data to the end of an existing append blob.
func (client Client) AppendBlock(ctx context.Context, accountName, containerName, blobName string, input AppendBlockInput) (result AppendBlockResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "AppendBlock", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "AppendBlock", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "AppendBlock", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "AppendBlock", "`blobName` cannot be an empty string.")
	}
	if input.Content != nil && len(*input.Content) > (4*1024*1024) {
		return result, validation.NewError("files.Client", "PutByteRange", "`input.Content` must be at most 4MB.")
	}

	req, err := client.AppendBlockPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "AppendBlock", nil, "Failure preparing request")
		return
	}

	resp, err := client.AppendBlockSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "AppendBlock", resp, "Failure sending request")
		return
	}

	result, err = client.AppendBlockResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "AppendBlock", resp, "Failure responding to request")
		return
	}

	return
}

// AppendBlockPreparer prepares the AppendBlock request.
func (client Client) AppendBlockPreparer(ctx context.Context, accountName, containerName, blobName string, input AppendBlockInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "appendblock"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.BlobConditionAppendPosition != nil {
		headers["x-ms-blob-condition-appendpos"] = *input.BlobConditionAppendPosition
	}
	if input.BlobConditionMaxSize != nil {
		headers["x-ms-blob-condition-maxsize"] = *input.BlobConditionMaxSize
	}
	if input.ContentMD5 != nil {
		headers["x-ms-blob-content-md5"] = *input.ContentMD5
	}
	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.Content != nil {
		headers["Content-Length"] = int(len(*input.Content))
	}

	decorators := []autorest.PrepareDecorator{
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers),
	}

	if input.Content != nil {
		decorators = append(decorators, autorest.WithBytes(input.Content))
	}

	preparer := autorest.CreatePreparer(decorators...)
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// AppendBlockSender sends the AppendBlock request. The method will close the
// http.Response Body if it receives an error.
func (client Client) AppendBlockSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// AppendBlockResponder handles the response to the AppendBlock request. The method always
// closes the http.Response Body.
func (client Client) AppendBlockResponder(resp *http.Response) (result AppendBlockResult, err error) {
	if resp != nil && resp.Header != nil {
		result.BlobAppendOffset = resp.Header.Get("x-ms-blob-append-offset")
		result.ContentMD5 = resp.Header.Get("ETag")
		result.ETag = resp.Header.Get("ETag")
		result.LastModified = resp.Header.Get("Last-Modified")

		if v := resp.Header.Get("x-ms-blob-committed-block-count"); v != "" {
			i, innerErr := strconv.Atoi(v)
			if innerErr != nil {
				err = fmt.Errorf("Error parsing %q as an integer: %s", v, innerErr)
				return
			}

			result.BlobCommittedBlockCount = int64(i)
		}
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
