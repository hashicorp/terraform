package scaleway

import (
	"github.com/scaleway/scaleway-cli/pkg/api"
	"github.com/scaleway/scaleway-cli/pkg/scwversion"
)

type Config struct {
	Organization string
	ApiKey       string
}

type Client struct {
	scaleway *api.ScalewayAPI
}

func (c *Config) Client() (*Client, error) {
	api, err := api.NewScalewayAPI(
		c.Organization,
		c.ApiKey,
		scwversion.UserAgent(),
	)
	if err != nil {
		return nil, err
	}
	return &Client{api}, nil
}
