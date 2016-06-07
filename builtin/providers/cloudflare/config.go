package cloudflare

import (
	"log"

	// NOTE: Temporary until they merge my PR:
	"github.com/mitchellh/cloudflare-go"
)

type Config struct {
	Email string
	Token string
}

// Client() returns a new client for accessing cloudflare.
func (c *Config) Client() (*cloudflare.API, error) {
	client := cloudflare.New(c.Token, c.Email)
	log.Printf("[INFO] CloudFlare Client configured for user: %s", c.Email)
	return client, nil
}
