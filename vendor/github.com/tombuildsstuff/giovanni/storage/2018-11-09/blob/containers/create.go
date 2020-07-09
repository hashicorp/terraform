package containers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
	"github.com/tombuildsstuff/giovanni/storage/internal/metadata"
)

type CreateInput struct {
	// Specifies whether data in the container may be accessed publicly and the level of access
	AccessLevel AccessLevel

	// A name-value pair to associate with the container as metadata.
	MetaData map[string]string
}

type CreateResponse struct {
	autorest.Response
	Error *ErrorResponse `xml:"Error"`
}

// Create creates a new container under the specified account.
// If the container with the same name already exists, the operation fails.
func (client Client) Create(ctx context.Context, accountName, containerName string, input CreateInput) (result CreateResponse, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "Create", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "Create", "`containerName` cannot be an empty string.")
	}
	if err := metadata.Validate(input.MetaData); err != nil {
		return result, validation.NewError("containers.Client", "Create", fmt.Sprintf("`input.MetaData` is not valid: %s.", err))
	}

	req, err := client.CreatePreparer(ctx, accountName, containerName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "Create", nil, "Failure preparing request")
		return
	}

	resp, err := client.CreateSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "Create", resp, "Failure sending request")
		return
	}

	result, err = client.CreateResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "Create", resp, "Failure responding to request")
		return
	}

	return
}

// CreatePreparer prepares the Create request.
func (client Client) CreatePreparer(ctx context.Context, accountName string, containerName string, input CreateInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"restype": autorest.Encode("path", "container"),
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
	}

	headers = client.setAccessLevelIntoHeaders(headers, input.AccessLevel)
	headers = metadata.SetIntoHeaders(headers, input.MetaData)

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/xml; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(endpoints.GetBlobEndpoint(client.BaseURI, accountName)),
		autorest.WithPathParameters("/{containerName}", pathParameters),
		autorest.WithQueryParameters(queryParameters),
		autorest.WithHeaders(headers))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateSender sends the Create request. The method will close the
// http.Response Body if it receives an error.
func (client Client) CreateSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// CreateResponder handles the response to the Create request. The method always
// closes the http.Response Body.
func (client Client) CreateResponder(resp *http.Response) (result CreateResponse, err error) {
	successfulStatusCodes := []int{
		http.StatusCreated,
	}
	if autorest.ResponseHasStatusCode(resp, successfulStatusCodes...) {
		// when successful there's no response
		err = autorest.Respond(
			resp,
			client.ByInspecting(),
			azure.WithErrorUnlessStatusCode(successfulStatusCodes...),
			autorest.ByClosing())
		result.Response = autorest.Response{Response: resp}
	} else {
		// however when there's an error the error's in the response
		err = autorest.Respond(
			resp,
			client.ByInspecting(),
			azure.WithErrorUnlessStatusCode(successfulStatusCodes...),
			autorest.ByUnmarshallingXML(&result),
			autorest.ByClosing())
		result.Response = autorest.Response{Response: resp}
	}

	return
}
