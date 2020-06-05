package blobs

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type GetSnapshotPropertiesInput struct {
	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string

	// The ID of the Snapshot which should be retrieved
	SnapshotID string
}

// GetSnapshotProperties returns all user-defined metadata, standard HTTP properties, and system properties for
// the specified snapshot of a blob
func (client Client) GetSnapshotProperties(ctx context.Context, accountName, containerName, blobName string, input GetSnapshotPropertiesInput) (result GetPropertiesResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "GetSnapshotProperties", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "GetSnapshotProperties", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "GetSnapshotProperties", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "GetSnapshotProperties", "`blobName` cannot be an empty string.")
	}
	if input.SnapshotID == "" {
		return result, validation.NewError("blobs.Client", "GetSnapshotProperties", "`input.SnapshotID` cannot be an empty string.")
	}

	req, err := client.GetSnapshotPropertiesPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetSnapshotProperties", nil, "Failure preparing request")
		return
	}

	// we re-use the GetProperties methods since this is otherwise the same
	resp, err := client.GetPropertiesSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetSnapshotProperties", resp, "Failure sending request")
		return
	}

	result, err = client.GetPropertiesResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "GetSnapshotProperties", resp, "Failure responding to request")
		return
	}

	return
}

// GetSnapshotPreparer prepares the GetSnapshot request.
func (client Client) GetSnapshotPropertiesPreparer(ctx context.Context, accountName, containerName, blobName string, input GetSnapshotPropertiesInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"snapshot": autorest.Encode("query", input.SnapshotID),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsHead(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}
