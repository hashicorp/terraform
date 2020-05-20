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

type SetPropertiesInput struct {
	CacheControl         *string
	ContentType          *string
	ContentMD5           *string
	ContentEncoding      *string
	ContentLanguage      *string
	LeaseID              *string
	ContentDisposition   *string
	ContentLength        *int64
	SequenceNumberAction *SequenceNumberAction
	BlobSequenceNumber   *string
}

type SetPropertiesResult struct {
	autorest.Response

	BlobSequenceNumber string
	Etag               string
}

// SetProperties sets system properties on the blob.
func (client Client) SetProperties(ctx context.Context, accountName, containerName, blobName string, input SetPropertiesInput) (result SetPropertiesResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "SetProperties", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "SetProperties", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "SetProperties", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "SetProperties", "`blobName` cannot be an empty string.")
	}

	req, err := client.SetPropertiesPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetProperties", nil, "Failure preparing request")
		return
	}

	resp, err := client.SetPropertiesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetProperties", resp, "Failure sending request")
		return
	}

	result, err = client.SetPropertiesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetProperties", resp, "Failure responding to request")
		return
	}

	return
}

type SequenceNumberAction string

var (
	Increment SequenceNumberAction = "increment"
	Max       SequenceNumberAction = "max"
	Update    SequenceNumberAction = "update"
)

// SetPropertiesPreparer prepares the SetProperties request.
func (client Client) SetPropertiesPreparer(ctx context.Context, accountName, containerName, blobName string, input SetPropertiesInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "properties"),
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
	if input.ContentLength != nil {
		headers["x-ms-blob-content-length"] = *input.ContentLength
	}
	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.SequenceNumberAction != nil {
		headers["x-ms-sequence-number-action"] = string(*input.SequenceNumberAction)
	}
	if input.BlobSequenceNumber != nil {
		headers["x-ms-blob-sequence-number"] = *input.BlobSequenceNumber
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// SetPropertiesSender sends the SetProperties request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SetPropertiesSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SetPropertiesResponder handles the response to the SetProperties request. The method always
// closes the http.Response Body.
func (client Client) SetPropertiesResponder(resp *http.Response) (result SetPropertiesResult, err error) {
	if resp != nil && resp.Header != nil {
		result.BlobSequenceNumber = resp.Header.Get("x-ms-blob-sequence-number")
		result.Etag = resp.Header.Get("Etag")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
