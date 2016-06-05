package scaleway

import (
	"github.com/scaleway/scaleway-cli/pkg/api"
	"github.com/scaleway/scaleway-cli/pkg/scwversion"
)

// Config contains scaleway configuration values
type Config struct {
	Organization string
	APIKey       string
}

// Client contains scaleway api clients
type Client struct {
	scaleway *api.ScalewayAPI
}

// Client configures and returns a fully initialized Scaleway client
func (c *Config) Client() (*Client, error) {
	api, err := api.NewScalewayAPI(
		c.Organization,
		c.APIKey,
		scwversion.UserAgent(),
	)
	if err != nil {
		return nil, err
	}
	return &Client{api}, nil
}
