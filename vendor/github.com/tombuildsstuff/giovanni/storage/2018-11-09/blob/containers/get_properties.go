package containers

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

// GetProperties returns the properties for this Container without a Lease
func (client Client) GetProperties(ctx context.Context, accountName, containerName string) (ContainerProperties, error) {
	// If specified, Get Container Properties only succeeds if the container’s lease is active and matches this ID.
	// If there is no active lease or the ID does not match, 412 (Precondition Failed) is returned.
	return client.GetPropertiesWithLeaseID(ctx, accountName, containerName, "")
}

// GetPropertiesWithLeaseID returns the properties for this Container using the specified LeaseID
func (client Client) GetPropertiesWithLeaseID(ctx context.Context, accountName, containerName, leaseID string) (result ContainerProperties, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "GetPropertiesWithLeaseID", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "GetPropertiesWithLeaseID", "`containerName` cannot be an empty string.")
	}

	req, err := client.GetPropertiesWithLeaseIDPreparer(ctx, accountName, containerName, leaseID)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "GetProperties", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetPropertiesWithLeaseIDSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "GetProperties", resp, "Failure sending request")
		return
	}

	result, err = client.GetPropertiesWithLeaseIDResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "GetProperties", resp, "Failure responding to request")
		return
	}

	return
}

// GetPropertiesWithLeaseIDPreparer prepares the GetPropertiesWithLeaseID request.
func (client Client) GetPropertiesWithLeaseIDPreparer(ctx context.Context, accountName, containerName, leaseID string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"restype": autorest.Encode("path", "container"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	// If specified, Get Container Properties only succeeds if the container’s lease is active and matches this ID.
	// If there is no active lease or the ID does not match, 412 (Precondition Failed) is returned.
	if leaseID != "" {
		headers["x-ms-lease-id"] = leaseID
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/xml; charset=utf-8"),
		autorest.AsGet(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetPropertiesWithLeaseIDSender sends the GetPropertiesWithLeaseID request. The method will close the
// http.Response Body if it receives an error.
func (client Client) GetPropertiesWithLeaseIDSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetPropertiesWithLeaseIDResponder handles the response to the GetPropertiesWithLeaseID request. The method always
// closes the http.Response Body.
func (client Client) GetPropertiesWithLeaseIDResponder(resp *http.Response) (result ContainerProperties, err error) {
	if resp != nil {
		result.LeaseStatus = LeaseStatus(resp.Header.Get("x-ms-lease-status"))
		result.LeaseState = LeaseState(resp.Header.Get("x-ms-lease-state"))
		if result.LeaseStatus == Locked {
			duration := LeaseDuration(resp.Header.Get("x-ms-lease-duration"))
			result.LeaseDuration = &duration
		}

		// If this header is not returned in the response, the container is private to the account owner.
		accessLevel := resp.Header.Get("x-ms-blob-public-access")
		if accessLevel != "" {
			result.AccessLevel = AccessLevel(accessLevel)
		} else {
			result.AccessLevel = Private
		}

		// we can't necessarily use strconv.ParseBool here since this could be nil (only in some API versions)
		result.HasImmutabilityPolicy = strings.EqualFold(resp.Header.Get("x-ms-has-immutability-policy"), "true")
		result.HasLegalHold = strings.EqualFold(resp.Header.Get("x-ms-has-legal-hold"), "true")

		result.MetaData = metadata.ParseFromHeaders(resp.Header)
	}

	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
