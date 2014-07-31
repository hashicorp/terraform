package openstack

import (
	"github.com/rackspace/gophercloud"
	"log"
	"os"
)

type Config struct {
	ApiUrl          string `mapstructure:"url"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	TenantName      string `mapstructure:"tenantName"`
	ComputeEndpoint string `mapstructure:"computeEndpoint"`
}

type OpenstackClient struct {
	Config         *Config
	AccessProvider gophercloud.AccessProvider
}

// Client() returns a new client for accessing openstack.
//
func (c *Config) Client() (*OpenstackClient, error) {

	if v := os.Getenv("OPENSTACK_URL"); v != "" {
		c.ApiUrl = v
	}
	if v := os.Getenv("OPENSTACK_USER"); v != "" {
		c.User = v
	}
	if v := os.Getenv("OPENSTACK_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("OPENSTACK_TENANT_NAME"); v != "" {
		c.TenantName = v
	}

	accessProvider, err := gophercloud.Authenticate(
		c.ApiUrl,
		gophercloud.AuthOptions{
			Username:   c.User,
			Password:   c.Password,
			TenantName: c.TenantName,
		},
	)

	if err != nil {
		return nil, err
	}

	client := &OpenstackClient{c, accessProvider}

	log.Printf("[INFO] Openstack Client configured for user %s", client.Config.User)

	return client, nil
}
