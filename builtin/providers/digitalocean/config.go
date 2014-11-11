package digitalocean

import (
	"log"
	"os"

	"github.com/pearkes/digitalocean"
)

type Config struct {
	Token string `mapstructure:"token"`
}

// Client() returns a new client for accessing digital
// ocean.
//
func (c *Config) Client() (*digitalocean.Client, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("DIGITALOCEAN_TOKEN"); v != "" {
		c.Token = v
	}

	client, err := digitalocean.NewClient(c.Token)

	log.Printf("[INFO] DigitalOcean Client configured for URL: %s", client.URL)

	if err != nil {
		return nil, err
	}

	return client, nil
}
