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
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

type SnapshotInput struct {
	// The ID of the Lease
	// This must be specified if a Lease is present on the Blob, else a 403 is returned
	LeaseID *string

	// MetaData is a user-defined name-value pair associated with the blob.
	// If no name-value pairs are specified, the operation will copy the base blob metadata to the snapshot.
	// If one or more name-value pairs are specified, the snapshot is created with the specified metadata,
	// and metadata is not copied from the base blob.
	MetaData map[string]string

	// A DateTime value which will only snapshot the blob if it has been modified since the specified date/time
	// If the base blob has not been modified, the Blob service returns status code 412 (Precondition Failed).
	IfModifiedSince *string

	// A DateTime value which will only snapshot the blob if it has not been modified since the specified date/time
	// If the base blob has been modified, the Blob service returns status code 412 (Precondition Failed).
	IfUnmodifiedSince *string

	// An ETag value to snapshot the blob only if its ETag value matches the value specified.
	// If the values do not match, the Blob service returns status code 412 (Precondition Failed).
	IfMatch *string

	// An ETag value for this conditional header to snapshot the blob only if its ETag value
	// does not match the value specified.
	// If the values are identical, the Blob service returns status code 412 (Precondition Failed).
	IfNoneMatch *string
}

type SnapshotResult struct {
	autorest.Response

	// The ETag of the snapshot
	ETag string

	// A DateTime value that uniquely identifies the snapshot.
	// The value of this header indicates the snapshot version,
	// and may be used in subsequent requests to access the snapshot.
	SnapshotDateTime string
}

// Snapshot captures a Snapshot of a given Blob
func (client Client) Snapshot(ctx context.Context, accountName, containerName, blobName string, input SnapshotInput) (result SnapshotResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "Snapshot", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "Snapshot", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "Snapshot", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "Snapshot", "`blobName` cannot be an empty string.")
	}
	if err := metadata.Validate(input.MetaData); err != nil {
		return result, validation.NewError("blobs.Client", "Snapshot", fmt.Sprintf("`input.MetaData` is not valid: %s.", err))
	}

	req, err := client.SnapshotPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Snapshot", nil, "Failure preparing request")
		return
	}

	resp, err := client.SnapshotSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "Snapshot", resp, "Failure sending request")
		return
	}

	result, err = client.SnapshotResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Snapshot", resp, "Failure responding to request")
		return
	}

	return
}

// SnapshotPreparer prepares the Snapshot request.
func (client Client) SnapshotPreparer(ctx context.Context, accountName, containerName, blobName string, input SnapshotInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("query", "snapshot"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
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

	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// SnapshotSender sends the Snapshot request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SnapshotSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SnapshotResponder handles the response to the Snapshot request. The method always
// closes the http.Response Body.
func (client Client) SnapshotResponder(resp *http.Response) (result SnapshotResult, err error) {
	if resp != nil && resp.Header != nil {
		result.ETag = resp.Header.Get("ETag")
		result.SnapshotDateTime = resp.Header.Get("x-ms-snapshot")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusCreated),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
