package cloudflare

import (
	"fmt"
	"log"

	"github.com/cloudflare/cloudflare-go"
)

type Config struct {
	Email string
	Token string
}

// Client() returns a new client for accessing cloudflare.
func (c *Config) Client() (*cloudflare.API, error) {
	client, err := cloudflare.New(c.Token, c.Email)
	if err != nil {
		return nil, fmt.Errorf("Error creating new CloudFlare client: %s", err)
	}
	log.Printf("[INFO] CloudFlare Client configured for user: %s", c.Email)
	return client, nil
}
