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

// SetTier sets the tier on a blob.
func (client Client) SetTier(ctx context.Context, accountName, containerName, blobName string, tier AccessTier) (result autorest.Response, err error) {
	if accountName == "" {
		return result, validation.NewError("blobs.Client", "SetTier", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("blobs.Client", "SetTier", "`containerName` cannot be an empty string.")
	}
	if strings.ToLower(containerName) != containerName {
		return result, validation.NewError("blobs.Client", "SetTier", "`containerName` must be a lower-cased string.")
	}
	if blobName == "" {
		return result, validation.NewError("blobs.Client", "SetTier", "`blobName` cannot be an empty string.")
	}

	req, err := client.SetTierPreparer(ctx, accountName, containerName, blobName, tier)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetTier", nil, "Failure preparing request")
		return
	}

	resp, err := client.SetTierSender(req)
	if err != nil {
		result = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetTier", resp, "Failure sending request")
		return
	}

	result, err = client.SetTierResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "blobs.Client", "SetTier", resp, "Failure responding to request")
		return
	}

	return
}

// SetTierPreparer prepares the SetTier request.
func (client Client) SetTierPreparer(ctx context.Context, accountName, containerName, blobName string, tier AccessTier) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
		"blobName":      autorest.Encode("path", blobName),
	}

	queryParameters := map[string]interface{}{
		"comp": autorest.Encode("path", "tier"),
	}

	headers := map[string]interface{}{
		"x-ms-version":     APIVersion,
		"x-ms-access-tier": string(tier),
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}/{blobName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// SetTierSender sends the SetTier request. The method will close the
// http.Response Body if it receives an error.
func (client Client) SetTierSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// SetTierResponder handles the response to the SetTier request. The method always
// closes the http.Response Body.
func (client Client) SetTierResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result = autorest.Response{Response: resp}
	return
}
