package openstack

import (
	"github.com/rackspace/gophercloud"
	"log"
	"os"
	"strings"
)

type Config struct {
	Auth       string `mapstructure:"authUrl"`
	User       string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	ApiKey     string `mapstructure:"apiKey"`
	TenantId   string `mapstructure:"tenantId"`
	TenantName string `mapstructure:"tenantName"`
}

type OpenstackClient struct {
	Config         *Config
	AccessProvider gophercloud.AccessProvider
}

// Client() returns a new client for accessing openstack.
//
func (c *Config) Client() (*OpenstackClient, error) {

	if v := os.Getenv("OS_AUTH_URL"); v != "" {
		c.Auth = v
	}
	if v := os.Getenv("OS_USERNAME"); v != "" {
		c.User = v
	}
	if v := os.Getenv("OS_PASSWORD"); v != "" {
		c.Password = v
	}
	if v := os.Getenv("OS_TENANT_ID"); v != "" {
		c.TenantId = v
	}
	if v := os.Getenv("OS_TENANT_NAME"); v != "" {
		c.TenantName = v
	}

	// OpenStack's auto-generated openrc.sh files do not append the suffix
	// /tokens to the authentication URL. This ensures it is present when
	// specifying the URL.
	if strings.Contains(c.Auth, "://") && !strings.HasSuffix(c.Auth, "/tokens") {
		c.Auth += "/tokens"
	}

	accessProvider, err := gophercloud.Authenticate(
		c.Auth,
		gophercloud.AuthOptions{
			ApiKey:     c.ApiKey,
			Username:   c.User,
			Password:   c.Password,
			TenantName: c.TenantName,
			TenantId:   c.TenantId,
		},
	)

	if err != nil {
		return nil, err
	}

	client := &OpenstackClient{c, accessProvider}

	log.Printf("[INFO] Openstack Client configured for user %s", client.Config.User)

	return client, nil
}
