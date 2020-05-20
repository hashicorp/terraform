package blobs

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

// GetResourceID returns the Resource ID for the given Blob
// This can be useful when, for example, you're using this as a unique identifier
func (client Client) GetResourceID(accountName, containerName, blobName string) string {
	domain := endpoints.GetBlobEndpoint(client.BaseURI, accountName)
	return fmt.Sprintf("%s/%s/%s", domain, containerName, blobName)
}

type ResourceID struct {
	AccountName   string
	ContainerName string
	BlobName      string
}

// ParseResourceID parses the Resource ID and returns an object which can be used
// to interact with the Blob Resource
func ParseResourceID(id string) (*ResourceID, error) {
	// example: https://foo.blob.core.windows.net/Bar/example.vhd
	if id == "" {
		return nil, fmt.Errorf("`id` was empty")
	}

	uri, err := url.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("Error parsing ID as a URL: %s", err)
	}

	accountName, err := endpoints.GetAccountNameFromEndpoint(uri.Host)
	if err != nil {
		return nil, fmt.Errorf("Error parsing Account Name: %s", err)
	}

	path := strings.TrimPrefix(uri.Path, "/")
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return nil, fmt.Errorf("Expected the path to contain segments but got none")
	}

	containerName := segments[0]
	blobName := strings.TrimPrefix(path, containerName)
	blobName = strings.TrimPrefix(blobName, "/")
	return &ResourceID{
		AccountName:   *accountName,
		ContainerName: containerName,
		BlobName:      blobName,
	}, nil
}
