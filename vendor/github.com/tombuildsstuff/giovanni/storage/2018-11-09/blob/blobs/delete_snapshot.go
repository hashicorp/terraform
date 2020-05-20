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

type DeleteSnapshotInput struct {
	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string

	// The DateTime of the Snapshot which should be marked for Deletion
	SnapshotDateTime string
}

// DeleteSnapshot marks a single Snapshot of a Blob for Deletion based on it's DateTime, which will be deleted during the next Garbage Collection cycle.
func (client Client) DeleteSnapshot(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshot", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshot", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "DeleteSnapshot", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshot", "`blobName` cannot be an empty string.")
	}
	if input.SnapshotDateTime == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshot", "`input.SnapshotDateTime` cannot be an empty string.")
	}

	req, err := client.DeleteSnapshotPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshot", nil, "Failure preparing request")
		return
	}

	resp, err := client.DeleteSnapshotSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshot", resp, "Failure sending request")
		return
	}

	result, err = client.DeleteSnapshotResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshot", resp, "Failure responding to request")
		return
	}

	return
}

// DeleteSnapshotPreparer prepares the DeleteSnapshot request.
func (client Client) DeleteSnapshotPreparer(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"snapshot": autorest.Encode("query", input.SnapshotDateTime),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSnapshotSender sends the DeleteSnapshot request. The method will close the
// http.Response Body if it receives an error.
func (client Client) DeleteSnapshotSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// DeleteSnapshotResponder handles the response to the DeleteSnapshot request. The method always
// closes the http.Response Body.
func (client Client) DeleteSnapshotResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
