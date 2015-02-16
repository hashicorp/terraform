package rackspace

import (
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

	c.rsClient = client

	return nil
}

func (c *Config) computeClient(region string) (*gophercloud.ServiceClient, error) {
	return rackspace.NewComputeV2(c.rsClient, gophercloud.EndpointOpts{
		Region: region,
	})
}
