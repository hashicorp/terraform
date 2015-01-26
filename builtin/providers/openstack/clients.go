package openstack

import (
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
)

func newClient(c *Config, service, region string, version int) (*gophercloud.ServiceClient, error) {
	var serviceClient *gophercloud.ServiceClient
	switch service {
	case "compute":
		if version == 2 {
			serviceClient, err = openstack.NewComputeV2(c.osClient, gophercloud.EndpointOpts{
				Region: region,
			})
		}
	case "networking":
		if version == 2 {
			serviceClient, err = openstack.NewNetworkV2(c.osClient, gophercloud.EndpointOpts{
				Region: region,
			})
		}
	}
	return serviceClient, err
}
