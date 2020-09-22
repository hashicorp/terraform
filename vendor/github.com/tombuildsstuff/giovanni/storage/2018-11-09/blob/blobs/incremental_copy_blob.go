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

type IncrementalCopyBlobInput struct {
	CopySource        string
	IfModifiedSince   *string
	IfUnmodifiedSince *string
	IfMatch           *string
	IfNoneMatch       *string
}

// IncrementalCopyBlob copies a snapshot of the source page blob to a destination page blob.
// The snapshot is copied such that only the differential changes between the previously copied
// snapshot are transferred to the destination.
// The copied snapshots are complete copies of the original snapshot and can be read or copied from as usual.
func (client Client) IncrementalCopyBlob(ctx context.Context, accountName, containerName, blobName string, input IncrementalCopyBlobInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "IncrementalCopyBlob", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "IncrementalCopyBlob", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "IncrementalCopyBlob", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "IncrementalCopyBlob", "`blobName` cannot be an empty string.")
	}
	if input.CopySource == "" {
		return result, validation.NewError("blobs.Client", "IncrementalCopyBlob", "`input.CopySource` cannot be an empty string.")
	}

	req, err := client.IncrementalCopyBlobPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "IncrementalCopyBlob", nil, "Failure preparing request")
		return
	}

	resp, err := client.IncrementalCopyBlobSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "IncrementalCopyBlob", resp, "Failure sending request")
		return
	}

	result, err = client.IncrementalCopyBlobResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "IncrementalCopyBlob", resp, "Failure responding to request")
		return
	}

	return
}

// IncrementalCopyBlobPreparer prepares the IncrementalCopyBlob request.
func (client Client) IncrementalCopyBlobPreparer(ctx context.Context, accountName, containerName, blobName string, input IncrementalCopyBlobInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "incrementalcopy"),
	}

	headers := map[string]interface{}{
		"x-ms-version":     APIVersion,
		"x-ms-copy-source": input.CopySource,
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
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// IncrementalCopyBlobSender sends the IncrementalCopyBlob request. The method will close the
// http.Response Body if it receives an error.
func (client Client) IncrementalCopyBlobSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// IncrementalCopyBlobResponder handles the response to the IncrementalCopyBlob request. The method always
// closes the http.Response Body.
func (client Client) IncrementalCopyBlobResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
