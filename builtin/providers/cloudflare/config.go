package cloudflare

import (
	"log"

	"github.com/crackcomm/cloudflare"
)

type Config struct {
	Email string
	Token string
}

// Client() returns a new client for accessing cloudflare.
func (c *Config) Client() (*cloudflare.Client, error) {
	client := cloudflare.New(&cloudflare.Options{
		Email: c.Email,
		Key:   c.Token,
	})

	log.Printf("[INFO] CloudFlare Client configured for user: %s", c.Email)

	return client, nil
}
