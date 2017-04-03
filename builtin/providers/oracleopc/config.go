package opc

import (
	"fmt"
	"github.com/oracle/terraform-provider-compute/sdk/compute"
	"net/url"
)

type Config struct {
	User            string
	Password        string
	IdentityDomain  string
	Endpoint        string
	MaxRetryTimeout int
}

type storageAttachment struct {
	index int
	instanceName *compute.InstanceName
}

type OPCClient struct {
	*compute.AuthenticatedClient
	MaxRetryTimeout int
	storageAttachmentsByVolumeCache map[string][]storageAttachment
}

func (c *Config) Client() (*OPCClient, error) {
	u, err := url.ParseRequestURI(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Invalid endpoint URI: %s", err)
	}

	client := compute.NewComputeClient(c.IdentityDomain, c.User, c.Password, u)
	authenticatedClient, err := client.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("Authentication failed: %s", err)
	}

	opcClient := &OPCClient{
		AuthenticatedClient: authenticatedClient,
		MaxRetryTimeout:     c.MaxRetryTimeout,
		storageAttachmentsByVolumeCache: make(map[string][]storageAttachment),
	}

	return opcClient, nil
}
