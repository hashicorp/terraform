package sdc

import (
	"log"

	"github.com/kiasaki/go-sdc"
)

type Config struct {
	Url     string
	Account string
	User    string
	KeyId   string
	Key     string
}

// Client() returns a new client for accessing SDC.
func (c *Config) Client() (*sdc.Client, error) {
	client := sdc.NewClient(c.Url, c.Account, c.User, c.KeyId, c.Key)

	log.Printf("[INFO] SDC Client configured for URL: %s", client.Url)

	return client, nil
}
