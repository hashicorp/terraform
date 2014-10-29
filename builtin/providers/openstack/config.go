package openstack

import (
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

type Config struct {
	Region           string
	Username         string
	Password         string
	IdentityEndpoint string
	TenantName string

	computeV2Client *gophercloud.ServiceClient
}

func (c *Config) loadAndValidate() error {
	ao := gophercloud.AuthOptions{
		Username: c.Username,
		Password: c.Password,
		IdentityEndpoint: c.IdentityEndpoint,
		TenantName: c.TenantName,
	}

	client, err := openstack.AuthenticatedClient(ao)
	if err != nil {
		return err
	}

	c.computeV2Client, err = openstack.NewComputeV2(client, gophercloud.EndpointOpts{
		Region: c.Region,
	})
	if err != nil {
		return err
	}

	return nil
}
