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

type GetPageRangesInput struct {
	LeaseID *string

	StartByte *int64
	EndByte   *int64
}

type GetPageRangesResult struct {
	autorest.Response

	// The size of the blob in bytes
	ContentLength *int64

	// The Content Type of the blob
	ContentType string

	// The ETag associated with this blob
	ETag string

	PageRanges []PageRange `xml:"PageRange"`
}

type PageRange struct {
	// The start byte offset for this range, inclusive
	Start int64 `xml:"Start"`

	// The end byte offset for this range, inclusive
	End int64 `xml:"End"`
}

// GetPageRanges returns the list of valid page ranges for a page blob or snapshot of a page blob.
func (client Client) GetPageRanges(ctx context.Context, accountName, containerName, blobName string, input GetPageRangesInput) (result GetPageRangesResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "GetPageRanges", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "GetPageRanges", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "GetPageRanges", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "GetPageRanges", "`blobName` cannot be an empty string.")
	}
	if (input.StartByte != nil && input.EndByte == nil) || input.StartByte == nil && input.EndByte != nil {
		return result, validation.NewError("blobs.Client", "GetPageRanges", "`input.StartByte` and `input.EndByte` must both be specified, or both be nil.")
	}

	req, err := client.GetPageRangesPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetPageRanges", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetPageRangesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetPageRanges", resp, "Failure sending request")
		return
	}

	result, err = client.GetPageRangesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetPageRanges", resp, "Failure responding to request")
		return
	}

	return
}

// GetPageRangesPreparer prepares the GetPageRanges request.
func (client Client) GetPageRangesPreparer(ctx context.Context, accountName, containerName, blobName string, input GetPageRangesInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "pagelist"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	if input.StartByte != nil && input.EndByte != nil {
		headers["x-ms-range"] = fmt.Sprintf("bytes=%d-%d", *input.StartByte, *input.EndByte)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetPageRangesSender sends the GetPageRanges request. The method will close the
// http.Response Body if it receives an error.
func (client Client) GetPageRangesSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetPageRangesResponder handles the response to the GetPageRanges request. The method always
// closes the http.Response Body.
func (client Client) GetPageRangesResponder(resp *http.Response) (result GetPageRangesResult, err error) {
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
