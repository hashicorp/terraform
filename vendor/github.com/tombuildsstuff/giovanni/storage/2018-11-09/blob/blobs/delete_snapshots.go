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

type DeleteSnapshotsInput struct {
	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string
}

// DeleteSnapshots marks all Snapshots of a Blob for Deletion, which will be deleted during the next Garbage Collection Cycle.
func (client Client) DeleteSnapshots(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotsInput) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshots", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshots", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "DeleteSnapshots", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "DeleteSnapshots", "`blobName` cannot be an empty string.")
	}

	req, err := client.DeleteSnapshotsPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshots", nil, "Failure preparing request")
		return
	}

	resp, err := client.DeleteSnapshotsSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshots", resp, "Failure sending request")
		return
	}

	result, err = client.DeleteSnapshotsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "DeleteSnapshots", resp, "Failure responding to request")
		return
	}

	return
}

// DeleteSnapshotsPreparer prepares the DeleteSnapshots request.
func (client Client) DeleteSnapshotsPreparer(ctx context.Context, accountName, containerName, blobName string, input DeleteSnapshotsInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
		// only delete the snapshots but leave the blob as-is
		"x-ms-delete-snapshots": "only",
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSnapshotsSender sends the DeleteSnapshots request. The method will close the
// http.Response Body if it receives an error.
func (client Client) DeleteSnapshotsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// DeleteSnapshotsResponder handles the response to the DeleteSnapshots request. The method always
// closes the http.Response Body.
func (client Client) DeleteSnapshotsResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
