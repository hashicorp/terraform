package openstack

import (
	"log"
	"os"
)

type Config struct {
	ApiUrl   string `mapstructure:"url"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type OpenstackClient struct {
	Config *Config
	Token  string
}

type Client struct {
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

	//client, err := dnsimple.NewClient(c.Email, c.Token)
	client := &OpenstackClient{c, "abcd"}

	/*if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}*/

	log.Printf("[INFO] Openstack Client configured for user: %s", client.Config.User)

	return client, nil
}
