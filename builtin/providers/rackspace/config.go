package rackspace

import (
	"fmt"

	//tf "github.com/hashicorp/terraform"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/rackspace"
)

// Config represents the options a user can provide to authenticate to a
// Rackspace endpoing
type Config struct {
	Username         string
	UserID           string
	Password         string
	APIKey           string
	IdentityEndpoint string
	TenantID         string
	TenantName       string
	DomainID         string
	DomainName       string

	rsClient *gophercloud.ProviderClient
}

func (c *Config) loadAndValidate() error {
	ao := gophercloud.AuthOptions{
		Username:         c.Username,
		UserID:           c.UserID,
		Password:         c.Password,
		APIKey:           c.APIKey,
		IdentityEndpoint: c.IdentityEndpoint,
		TenantID:         c.TenantID,
		TenantName:       c.TenantName,
		DomainID:         c.DomainID,
		DomainName:       c.DomainName,
	}

	client, err := rackspace.AuthenticatedClient(ao)
	if err != nil {
		return err
	}
	//client.UserAgent.Prepend("terraform/" + tf.Version)
	fmt.Printf("user agent: %s\n", client.UserAgent.Join())

	c.rsClient = client

	return nil
}

func (c *Config) blockStorageClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewBlockStorageV1(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}

func (c *Config) computeClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewComputeV2(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}

func (c *Config) networkingClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewNetworkV2(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}
