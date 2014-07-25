package cloudflare

import (
	"fmt"
	"log"

	"github.com/pearkes/cloudflare"
)

type Config struct {
	Token string `mapstructure:"token"`
	Email string `mapstructure:"email"`
}

// Client() returns a new client for accessing cloudflare.
//
func (c *Config) Client() (*cloudflare.Client, error) {
	client, err := cloudflare.NewClient(c.Email, c.Token)

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] CloudFlare Client configured for user: %s", client.Email)

	return client, nil
}
