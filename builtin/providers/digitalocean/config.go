package digitalocean

import (
	"log"

	"github.com/pearkes/digitalocean"
)

type Config struct {
	Token string `mapstructure:"token"`
}

// Client() returns a new client for accessing digital
// ocean.
//
func (c *Config) Client() (*digitalocean.Client, error) {
	client, err := digitalocean.NewClient(c.Token)

	log.Printf("[INFO] DigitalOcean Client configured for URL: %s", client.URL)

	if err != nil {
		return nil, err
	}

	return client, nil
}
