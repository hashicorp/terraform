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

type BlockList struct {
	CommittedBlockIDs   []BlockID `xml:"Committed,omitempty"`
	UncommittedBlockIDs []BlockID `xml:"Uncommitted,omitempty"`
	LatestBlockIDs      []BlockID `xml:"Latest,omitempty"`
}

type BlockID struct {
	Value string `xml:",chardata"`
}

type PutBlockListInput struct {
	BlockList          BlockList
	CacheControl       *string
	ContentDisposition *string
	ContentEncoding    *string
	ContentLanguage    *string
	ContentMD5         *string
	ContentType        *string
	MetaData           map[string]string
	LeaseID            *string
}

type PutBlockListResult struct {
	autorest.Response

	ContentMD5   string
	ETag         string
	LastModified string
}

// PutBlockList writes a blob by specifying the list of block IDs that make up the blob.
// In order to be written as part of a blob, a block must have been successfully written
// to the server in a prior Put Block operation.
func (client Client) PutBlockList(ctx context.Context, accountName, containerName, blobName string, input PutBlockListInput) (result PutBlockListResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockList", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockList", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutBlockList", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutBlockList", "`blobName` cannot be an empty string.")
	}

	req, err := client.PutBlockListPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockList", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutBlockListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockList", resp, "Failure sending request")
		return
	}

	result, err = client.PutBlockListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutBlockList", resp, "Failure responding to request")
		return
	}

	return
}

// PutBlockListPreparer prepares the PutBlockList request.
func (client Client) PutBlockListPreparer(ctx context.Context, accountName, containerName, blobName string, input PutBlockListInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "blocklist"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
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
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers),
		autorest.WithXML(input.BlockList))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutBlockListSender sends the PutBlockList request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutBlockListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutBlockListResponder handles the response to the PutBlockList request. The method always
// closes the http.Response Body.
func (client Client) PutBlockListResponder(resp *http.Response) (result PutBlockListResult, err error) {
	if resp != nil && resp.Header != nil {
		result.ContentMD5 = resp.Header.Get("Content-MD5")
		result.ETag = resp.Header.Get("ETag")
		result.LastModified = resp.Header.Get("Last-Modified")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
