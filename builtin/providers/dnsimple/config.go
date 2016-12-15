package dnsimple

import (
	"log"

	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type Config struct {
	Email   string
	Account string
	Token   string
}

// Client() returns a new client for accessing dnsimple.
func (c *Config) Client() (*dnsimple.Client, error) {
	client := dnsimple.NewClient(dnsimple.NewOauthTokenCredentials(c.Token))
	client.BaseURL = "https://api.sandbox.dnsimple.com"

	log.Printf("[INFO] DNSimple Client configured for account: %s", c.Account)

	return client, nil
}
