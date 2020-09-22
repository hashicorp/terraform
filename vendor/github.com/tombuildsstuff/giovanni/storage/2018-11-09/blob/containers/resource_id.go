package containers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tombuildsstuff/giovanni/storage/internal/endpoints"
)

// GetResourceID returns the Resource ID for the given Container
// This can be useful when, for example, you're using this as a unique identifier
func (client Client) GetResourceID(accountName, containerName string) string {
	domain := endpoints.GetBlobEndpoint(client.BaseURI, accountName)
	return fmt.Sprintf("%s/%s", domain, containerName)
}

// GetResourceManagerResourceID returns the Resource Manager specific
// ResourceID for a specific Storage Container
func (client Client) GetResourceManagerResourceID(subscriptionID, resourceGroup, accountName, containerName string) string {
	fmtStr := "/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s"
	return fmt.Sprintf(fmtStr, subscriptionID, resourceGroup, accountName, containerName)
}

type ResourceID struct {
	AccountName   string
	ContainerName string
}

// ParseResourceID parses the Resource ID and returns an object which can be used
// to interact with the Container Resource
func ParseResourceID(id string) (*ResourceID, error) {
	// example: https://foo.blob.core.windows.net/Bar
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

	containerName := strings.TrimPrefix(uri.Path, "/")
	return &ResourceID{
		AccountName:   *accountName,
		ContainerName: containerName,
	}, nil
}
