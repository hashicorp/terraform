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

type GetInput struct {
	LeaseID   *string
	StartByte *int64
	EndByte   *int64
}

type GetResult struct {
	autorest.Response

	Contents []byte
}

// Get reads or downloads a blob from the system, including its metadata and properties.
func (client Client) Get(ctx context.Context, accountName, containerName, blobName string, input GetInput) (result GetResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "Get", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "Get", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "Get", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "Get", "`blobName` cannot be an empty string.")
	}
	if input.LeaseID != nil && *input.LeaseID == "" {
		return result, validation.NewError("blobs.Client", "Get", "`input.LeaseID` should either be specified or nil, not an empty string.")
	}
	if (input.StartByte != nil && input.EndByte == nil) || input.StartByte == nil && input.EndByte != nil {
		return result, validation.NewError("blobs.Client", "Get", "`input.StartByte` and `input.EndByte` must both be specified, or both be nil.")
	}

	req, err := client.GetPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Get", resp, "Failure responding to request")
		return
	}

	return
}

// GetPreparer prepares the Get request.
func (client Client) GetPreparer(ctx context.Context, accountName, containerName, blobName string, input GetInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.StartByte != nil && input.EndByte != nil {
		headers["x-ms-range"] = fmt.Sprintf("bytes=%d-%d", *input.StartByte, *input.EndByte)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client Client) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client Client) GetResponder(resp *http.Response) (result GetResult, err error) {
	if resp != nil {
		result.Contents = make([]byte, resp.ContentLength)
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusPartialContent),
		autorest.ByUnmarshallingBytes(&result.Contents),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
