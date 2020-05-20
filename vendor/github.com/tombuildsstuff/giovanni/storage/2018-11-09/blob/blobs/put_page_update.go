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
)

type PutPageUpdateInput struct {
	StartByte int64
	EndByte   int64
	Content   []byte

	IfSequenceNumberEQ *string
	IfSequenceNumberLE *string
	IfSequenceNumberLT *string
	IfModifiedSince    *string
	IfUnmodifiedSince  *string
	IfMatch            *string
	IfNoneMatch        *string
	LeaseID            *string
}

type PutPageUpdateResult struct {
	autorest.Response

	BlobSequenceNumber string
	ContentMD5         string
	LastModified       string
}

// PutPageUpdate writes a range of pages to a page blob.
func (client Client) PutPageUpdate(ctx context.Context, accountName, containerName, blobName string, input PutPageUpdateInput) (result PutPageUpdateResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`blobName` cannot be an empty string.")
	}
	if input.StartByte < 0 {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`input.StartByte` must be greater than or equal to 0.")
	}
	if input.EndByte <= 0 {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", "`input.EndByte` must be greater than 0.")
	}

	expectedSize := (input.EndByte - input.StartByte) + 1
	actualSize := int64(len(input.Content))
	if expectedSize != actualSize {
		return result, validation.NewError("blobs.Client", "PutPageUpdate", fmt.Sprintf("Content Size was defined as %d but got %d.", expectedSize, actualSize))
	}

	req, err := client.PutPageUpdatePreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageUpdate", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutPageUpdateSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageUpdate", resp, "Failure sending request")
		return
	}

	result, err = client.PutPageUpdateResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageUpdate", resp, "Failure responding to request")
		return
	}

	return
}

// PutPageUpdatePreparer prepares the PutPageUpdate request.
func (client Client) PutPageUpdatePreparer(ctx context.Context, accountName, containerName, blobName string, input PutPageUpdateInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "page"),
	}

	headers := map[string]interface{}{
		"x-ms-version":    APIVersion,
		"x-ms-page-write": "update",
		"x-ms-range":      fmt.Sprintf("bytes=%d-%d", input.StartByte, input.EndByte),
		"Content-Length":  int(len(input.Content)),
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.IfSequenceNumberEQ != nil {
		headers["x-ms-if-sequence-number-eq"] = *input.IfSequenceNumberEQ
	}
	if input.IfSequenceNumberLE != nil {
		headers["x-ms-if-sequence-number-le"] = *input.IfSequenceNumberLE
	}
	if input.IfSequenceNumberLT != nil {
		headers["x-ms-if-sequence-number-lt"] = *input.IfSequenceNumberLT
	}
	if input.IfModifiedSince != nil {
		headers["If-Modified-Since"] = *input.IfModifiedSince
	}
	if input.IfUnmodifiedSince != nil {
		headers["If-Unmodified-Since"] = *input.IfUnmodifiedSince
	}
	if input.IfMatch != nil {
		headers["If-Match"] = *input.IfMatch
	}
	if input.IfNoneMatch != nil {
		headers["If-None-Match"] = *input.IfNoneMatch
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

// PutPageUpdateSender sends the PutPageUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutPageUpdateSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutPageUpdateResponder handles the response to the PutPageUpdate request. The method always
// closes the http.Response Body.
func (client Client) PutPageUpdateResponder(resp *http.Response) (result PutPageUpdateResult, err error) {
	if resp != nil && resp.Header != nil {
		result.BlobSequenceNumber = resp.Header.Get("x-ms-blob-sequence-number")
		result.ContentMD5 = resp.Header.Get("Content-MD5")
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
