package blobs

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// Client is the base client for Blob Storage Blobs.
type Client struct {
	autorest.Client
	BaseURI string
}

// New creates an instance of the Client client.
func New() Client {
	return NewWithEnvironment(azure.PublicCloud)
}

// NewWithBaseURI creates an instance of the Client client.
func NewWithEnvironment(environment azure.Environment) Client {
	return Client{
		Client:  autorest.NewClientWithUserAgent(UserAgent()),
		BaseURI: environment.StorageEndpointSuffix,
	}
}
