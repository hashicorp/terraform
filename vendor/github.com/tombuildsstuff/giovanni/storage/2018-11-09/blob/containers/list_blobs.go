package containers

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

type ListBlobsInput struct {
	Delimiter  *string
	Include    *[]Dataset
	Marker     *string
	MaxResults *int
	Prefix     *string
}

type ListBlobsResult struct {
	autorest.Response

	Delimiter  string  `xml:"Delimiter"`
	Marker     string  `xml:"Marker"`
	MaxResults int     `xml:"MaxResults"`
	NextMarker *string `xml:"NextMarker,omitempty"`
	Prefix     string  `xml:"Prefix"`
	Blobs      Blobs   `xml:"Blobs"`
}

type Blobs struct {
	Blobs      []BlobDetails `xml:"Blob"`
	BlobPrefix *BlobPrefix   `xml:"BlobPrefix"`
}

type BlobDetails struct {
	Name       string                 `xml:"Name"`
	Deleted    bool                   `xml:"Deleted,omitempty"`
	MetaData   map[string]interface{} `map:"Metadata,omitempty"`
	Properties *BlobProperties        `xml:"Properties,omitempty"`
	Snapshot   *string                `xml:"Snapshot,omitempty"`
}

type BlobProperties struct {
	AccessTier             *string `xml:"AccessTier,omitempty"`
	AccessTierInferred     *bool   `xml:"AccessTierInferred,omitempty"`
	AccessTierChangeTime   *string `xml:"AccessTierChangeTime,omitempty"`
	BlobType               *string `xml:"BlobType,omitempty"`
	BlobSequenceNumber     *string `xml:"x-ms-blob-sequence-number,omitempty"`
	CacheControl           *string `xml:"Cache-Control,omitempty"`
	ContentEncoding        *string `xml:"ContentEncoding,omitempty"`
	ContentLanguage        *string `xml:"Content-Language,omitempty"`
	ContentLength          *int64  `xml:"Content-Length,omitempty"`
	ContentMD5             *string `xml:"Content-MD5,omitempty"`
	ContentType            *string `xml:"Content-Type,omitempty"`
	CopyCompletionTime     *string `xml:"CopyCompletionTime,omitempty"`
	CopyId                 *string `xml:"CopyId,omitempty"`
	CopyStatus             *string `xml:"CopyStatus,omitempty"`
	CopySource             *string `xml:"CopySource,omitempty"`
	CopyProgress           *string `xml:"CopyProgress,omitempty"`
	CopyStatusDescription  *string `xml:"CopyStatusDescription,omitempty"`
	CreationTime           *string `xml:"CreationTime,omitempty"`
	ETag                   *string `xml:"Etag,omitempty"`
	DeletedTime            *string `xml:"DeletedTime,omitempty"`
	IncrementalCopy        *bool   `xml:"IncrementalCopy,omitempty"`
	LastModified           *string `xml:"Last-Modified,omitempty"`
	LeaseDuration          *string `xml:"LeaseDuration,omitempty"`
	LeaseState             *string `xml:"LeaseState,omitempty"`
	LeaseStatus            *string `xml:"LeaseStatus,omitempty"`
	RemainingRetentionDays *string `xml:"RemainingRetentionDays,omitempty"`
	ServerEncrypted        *bool   `xml:"ServerEncrypted,omitempty"`
}

type BlobPrefix struct {
	Name string `xml:"Name"`
}

// ListBlobs lists the blobs matching the specified query within the specified Container
func (client Client) ListBlobs(ctx context.Context, accountName, containerName string, input ListBlobsInput) (result ListBlobsResult, err error) {
	if accountName == "" {
		return result, validation.NewError("containers.Client", "ListBlobs", "`accountName` cannot be an empty string.")
	}
	if containerName == "" {
		return result, validation.NewError("containers.Client", "ListBlobs", "`containerName` cannot be an empty string.")
	}
	if input.MaxResults != nil && (*input.MaxResults <= 0 || *input.MaxResults > 5000) {
		return result, validation.NewError("containers.Client", "ListBlobs", "`input.MaxResults` can either be nil or between 0 and 5000.")
	}

	req, err := client.ListBlobsPreparer(ctx, accountName, containerName, input)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "ListBlobs", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListBlobsSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containers.Client", "ListBlobs", resp, "Failure sending request")
		return
	}

	result, err = client.ListBlobsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containers.Client", "ListBlobs", resp, "Failure responding to request")
		return
	}

	return
}

// ListBlobsPreparer prepares the ListBlobs request.
func (client Client) ListBlobsPreparer(ctx context.Context, accountName, containerName string, input ListBlobsInput) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"containerName": autorest.Encode("path", containerName),
	}

	queryParameters := map[string]interface{}{
		"comp":    autorest.Encode("query", "list"),
		"restype": autorest.Encode("query", "container"),
	}

	if input.Delimiter != nil {
		queryParameters["delimiter"] = autorest.Encode("query", *input.Delimiter)
	}
	if input.Include != nil {
		vals := make([]string, 0)
		for _, v := range *input.Include {
			vals = append(vals, string(v))
		}
		include := strings.Join(vals, ",")
		queryParameters["include"] = autorest.Encode("query", include)
	}
	if input.Marker != nil {
		queryParameters["marker"] = autorest.Encode("query", *input.Marker)
	}
	if input.MaxResults != nil {
		queryParameters["maxresults"] = autorest.Encode("query", *input.MaxResults)
	}
	if input.Prefix != nil {
		queryParameters["prefix"] = autorest.Encode("query", *input.Prefix)
	}

	headers := map[string]interface{}{
		"x-ms-version": APIVersion,
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

// ListBlobsSender sends the ListBlobs request. The method will close the
// http.Response Body if it receives an error.
func (client Client) ListBlobsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListBlobsResponder handles the response to the ListBlobs request. The method always
// closes the http.Response Body.
func (client Client) ListBlobsResponder(resp *http.Response) (result ListBlobsResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingXML(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}

	return
}
