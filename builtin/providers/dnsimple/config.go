package dnsimple

import (
	"fmt"
	"log"
	"os"

	"github.com/pearkes/dnsimple"
)

type Config struct {
	Token string `mapstructure:"token"`
	Email string `mapstructure:"email"`
}

// Client() returns a new client for accessing dnsimple.
//
func (c *Config) Client() (*dnsimple.Client, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("DNSIMPLE_EMAIL"); v != "" {
		c.Email = v
	}
	if v := os.Getenv("DNSIMPLE_TOKEN"); v != "" {
		c.Token = v
	}

	client, err := dnsimple.NewClient(c.Email, c.Token)

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] DNSimple Client configured for user: %s", client.Email)

	return client, nil
}
