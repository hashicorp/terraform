package blobs

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

type CopyInput struct {
	// Specifies the name of the source blob or file.
	// Beginning with version 2012-02-12, this value may be a URL of up to 2 KB in length that specifies a blob.
	// The value should be URL-encoded as it would appear in a request URI.
	// A source blob in the same storage account can be authenticated via Shared Key.
	// However, if the source is a blob in another account,
	// the source blob must either be public or must be authenticated via a shared access signature.
	// If the source blob is public, no authentication is required to perform the copy operation.
	//
	// Beginning with version 2015-02-21, the source object may be a file in the Azure File service.
	// If the source object is a file that is to be copied to a blob, then the source file must be authenticated
	// using a shared access signature, whether it resides in the same account or in a different account.
	//
	// Only storage accounts created on or after June 7th, 2012 allow the Copy Blob operation to
	// copy from another storage account.
	CopySource string

	// The ID of the Lease
	// Required if the destination blob has an active lease.
	// The lease ID specified for this header must match the lease ID of the destination blob.
	// If the request does not include the lease ID or it is not valid,
	// the operation fails with status code 412 (Precondition Failed).
	//
	// If this header is specified and the destination blob does not currently have an active lease,
	// the operation will also fail with status code 412 (Precondition Failed).
	LeaseID *string

	// The ID of the Lease on the Source Blob
	// Specify to perform the Copy Blob operation only if the lease ID matches the active lease ID of the source blob.
	SourceLeaseID *string

	// For page blobs on a premium account only. Specifies the tier to be set on the target blob
	AccessTier *AccessTier

	// A user-defined name-value pair associated with the blob.
	// If no name-value pairs are specified, the operation will copy the metadata from the source blob or
	// file to the destination blob.
	// If one or more name-value pairs are specified, the destination blob is created with the specified metadata,
	// and metadata is not copied from the source blob or file.
	MetaData map[string]string

	// An ETag value.
	// Specify an ETag value for this conditional header to copy the blob only if the specified
	// ETag value matches the ETag value for an existing destination blob.
	// If the ETag for the destination blob does not match the ETag specified for If-Match,
	// the Blob service returns status code 412 (Precondition Failed).
	IfMatch *string

	// An ETag value, or the wildcard character (*).
	// Specify an ETag value for this conditional header to copy the blob only if the specified
	// ETag value does not match the ETag value for the destination blob.
	// Specify the wildcard character (*) to perform the operation only if the destination blob does not exist.
	// If the specified condition isn't met, the Blob service returns status code 412 (Precondition Failed).
	IfNoneMatch *string

	// A DateTime value.
	// Specify this conditional header to copy the blob only if the destination blob
	// has been modified since the specified date/time.
	// If the destination blob has not been modified, the Blob service returns status code 412 (Precondition Failed).
	IfModifiedSince *string

	// A DateTime value.
	// Specify this conditional header to copy the blob only if the destination blob
	// has not been modified since the specified date/time.
	// If the destination blob has been modified, the Blob service returns status code 412 (Precondition Failed).
	IfUnmodifiedSince *string

	// An ETag value.
	// Specify this conditional header to copy the source blob only if its ETag matches the value specified.
	// If the ETag values do not match, the Blob service returns status code 412 (Precondition Failed).
	// This cannot be specified if the source is an Azure File.
	SourceIfMatch *string

	// An ETag value.
	// Specify this conditional header to copy the blob only if its ETag does not match the value specified.
	// If the values are identical, the Blob service returns status code 412 (Precondition Failed).
	// This cannot be specified if the source is an Azure File.
	SourceIfNoneMatch *string

	// A DateTime value.
	// Specify this conditional header to copy the blob only if the source blob has been modified
	// since the specified date/time.
	// If the source blob has not been modified, the Blob service returns status code 412 (Precondition Failed).
	// This cannot be specified if the source is an Azure File.
	SourceIfModifiedSince *string

	// A DateTime value.
	// Specify this conditional header to copy the blob only if the source blob has not been modified
	// since the specified date/time.
	// If the source blob has been modified, the Blob service returns status code 412 (Precondition Failed).
	// This header cannot be specified if the source is an Azure File.
	SourceIfUnmodifiedSince *string
}

type CopyResult struct {
	autorest.Response

	CopyID     string
	CopyStatus string
}

// Copy copies a blob to a destination within the storage account asynchronously.
func (client Client) Copy(ctx context.Context, accountName, containerName, blobName string, input CopyInput) (result CopyResult, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "Copy", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "Copy", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "Copy", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "Copy", "`blobName` cannot be an empty string.")
	}
	if input.CopySource == "" {
		return result, validation.NewError("blobs.Client", "Copy", "`input.CopySource` cannot be an empty string.")
	}

	req, err := client.CopyPreparer(ctx, accountName, containerName, blobName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Copy", nil, "Failure preparing request")
		return
	}

	resp, err := client.CopySender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "Copy", resp, "Failure sending request")
		return
	}

	result, err = client.CopyResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "Copy", resp, "Failure responding to request")
		return
	}

	return
}

// CopyPreparer prepares the Copy request.
func (client Client) CopyPreparer(ctx context.Context, accountName, containerName, blobName string, input CopyInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	headers := map[string]interface{}{
		"x-ms-version":     APIVersion,
		"x-ms-copy-source": autorest.Encode("header", input.CopySource),
	}

	if input.LeaseID != nil {
		headers["x-ms-lease-id"] = *input.LeaseID
	}
	if input.SourceLeaseID != nil {
		headers["x-ms-source-lease-id"] = *input.SourceLeaseID
	}
	if input.AccessTier != nil {
		headers["x-ms-access-tier"] = string(*input.AccessTier)
	}

	if input.IfMatch != nil {
		headers["If-Match"] = *input.IfMatch
	}
	if input.IfNoneMatch != nil {
		headers["If-None-Match"] = *input.IfNoneMatch
	}
	if input.IfUnmodifiedSince != nil {
		headers["If-Unmodified-Since"] = *input.IfUnmodifiedSince
	}
	if input.IfModifiedSince != nil {
		headers["If-Modified-Since"] = *input.IfModifiedSince
	}

	if input.SourceIfMatch != nil {
		headers["x-ms-source-if-match"] = *input.SourceIfMatch
	}
	if input.SourceIfNoneMatch != nil {
		headers["x-ms-source-if-none-match"] = *input.SourceIfNoneMatch
	}
	if input.SourceIfModifiedSince != nil {
		headers["x-ms-source-if-modified-since"] = *input.SourceIfModifiedSince
	}
	if input.SourceIfUnmodifiedSince != nil {
		headers["x-ms-source-if-unmodified-since"] = *input.SourceIfUnmodifiedSince
	}

	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CopySender sends the Copy request. The method will close the
// http.Response Body if it receives an error.
func (client Client) CopySender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// CopyResponder handles the response to the Copy request. The method always
// closes the http.Response Body.
func (client Client) CopyResponder(resp *http.Response) (result CopyResult, err error) {
	if resp != nil && resp.Header != nil {
		result.CopyID = resp.Header.Get("x-ms-copy-id")
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusAccepted),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
