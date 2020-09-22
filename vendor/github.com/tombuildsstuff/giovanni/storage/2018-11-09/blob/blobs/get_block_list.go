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

type GetBlockListInput struct {
	BlockListType BlockListType
	LeaseID       *string
}

type GetBlockListResult struct {
	autorest.Response

	// The size of the blob in bytes
	ContentLength *int64

	// The Content Type of the blob
	ContentType string

	// The ETag associated with this blob
	ETag string

	// A list of blocks which have been committed
	CommittedBlocks CommittedBlocks `xml:"CommittedBlocks,omitempty"`

	// A list of blocks which have not yet been committed
	UncommittedBlocks UncommittedBlocks `xml:"UncommittedBlocks,omitempty"`
}

// GetBlockList retrieves the list of blocks that have been uploaded as part of a block blob.
func (client Client) GetBlockList(ctx context.Context, accountName, containerName, blobName string, input GetBlockListInput) (result GetBlockListResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "GetBlockList", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "GetBlockList", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "GetBlockList", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "GetBlockList", "`blobName` cannot be an empty string.")
	}

	req, err := client.GetBlockListPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetBlockList", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetBlockListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetBlockList", resp, "Failure sending request")
		return
	}

	result, err = client.GetBlockListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetBlockList", resp, "Failure responding to request")
		return
	}

	return
}

// GetBlockListPreparer prepares the GetBlockList request.
func (client Client) GetBlockListPreparer(ctx context.Context, accountName, containerName, blobName string, input GetBlockListInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"blocklisttype": autorest.Encode("query", string(input.BlockListType)),
		"comp":          autorest.Encode("query", "blocklist"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetBlockListSender sends the GetBlockList request. The method will close the
// http.Response Body if it receives an error.
func (client Client) GetBlockListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetBlockListResponder handles the response to the GetBlockList request. The method always
// closes the http.Response Body.
func (client Client) GetBlockListResponder(resp *http.Response) (result GetBlockListResult, err error) {
	if resp != nil && resp.Header != nil {
		result.ContentType = resp.Header.Get("Content-Type")
		result.ETag = resp.Header.Get("ETag")

		if v := resp.Header.Get("x-ms-blob-content-length"); v != "" {
			i, innerErr := strconv.Atoi(v)
			if innerErr != nil {
				err = fmt.Errorf("Error parsing %q as an integer: %s", v, innerErr)
				return
			}

			i64 := int64(i)
			result.ContentLength = &i64
		}
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingXML(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
