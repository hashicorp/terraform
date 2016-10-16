package nomad

// implementation of issue 4442

import (
	"github.com/hashicorp/nomad/api"
)

// Config contains nomad configuration values
type Config struct {
	Address  string
	Region   string
	Username string
	Password string
}

// Client contains nomad api clients
type Client struct {
	nomad *api.Client
}

// Client configures and returns a fully initialized nomad client
func (c *Config) Client() (*Client, error) {
	config := api.Config{
		Address: c.Address,
		Region:  c.Region,
		HttpAuth: &api.HttpBasicAuth{
			Username: c.Username,
			Password: c.Password,
		},
	}
	client, err := api.NewClient(&config)
	if err != nil {
		return nil, err
	}
	return &Client{client}, nil
}
