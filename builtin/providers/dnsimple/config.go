package dnsimple

import (
	"log"
	"os"

	"github.com/rubyist/go-dnsimple"
)

type Config struct {
	Token string `mapstructure:"token"`
	Email string `mapstructure:"email"`
}

// Client() returns a new client for accessing heroku.
//
func (c *Config) Client() (*dnsimple.DNSimpleClient, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("DNSIMPLE_EMAIL"); v != "" {
		c.Email = v
	}
	if v := os.Getenv("DNSIMPLE_TOKEN"); v != "" {
		c.Token = v
	}

	client := dnsimple.NewClient(c.Token, c.Email)

	log.Printf("[INFO] DNSimple Client configured for user: %s", client.Email)

	return client, nil
}
