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

type PutPageClearInput struct {
	StartByte int64
	EndByte   int64

	LeaseID *string
}

// PutPageClear clears a range of pages within a page blob.
func (client Client) PutPageClear(ctx context.Context, accountName, containerName, blobName string, input PutPageClearInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`blobName` cannot be an empty string.")
	}
	if input.StartByte < 0 {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`input.StartByte` must be greater than or equal to 0.")
	}
	if input.EndByte <= 0 {
		return result, validation.NewError("blobs.Client", "PutPageClear", "`input.EndByte` must be greater than 0.")
	}

	req, err := client.PutPageClearPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageClear", nil, "Failure preparing request")
		return
	}

	resp, err := client.PutPageClearSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageClear", resp, "Failure sending request")
		return
	}

	result, err = client.PutPageClearResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "PutPageClear", resp, "Failure responding to request")
		return
	}

	return
}

// PutPageClearPreparer prepares the PutPageClear request.
func (client Client) PutPageClearPreparer(ctx context.Context, accountName, containerName, blobName string, input PutPageClearInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "page"),
	}

	headers := map[string]interface{}{
		"x-ms-version":    APIVersion,
		"x-ms-page-write": "clear",
		"x-ms-range":      fmt.Sprintf("bytes=%d-%d", input.StartByte, input.EndByte),
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// PutPageClearSender sends the PutPageClear request. The method will close the
// http.Response Body if it receives an error.
func (client Client) PutPageClearSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// PutPageClearResponder handles the response to the PutPageClear request. The method always
// closes the http.Response Body.
func (client Client) PutPageClearResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}

	return
}
