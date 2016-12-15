package dnsimple

import (
	"log"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/hashicorp/terraform/terraform"
)

type Config struct {
	Email   string
	Account string
	Token   string
}

// Client represents the DNSimple provider client.
// This is a convenient container for the configuration and the underlying API client.
type Client struct {
	client *dnsimple.Client
	config *Config
}

// Client() returns a new client for accessing dnsimple.
func (c *Config) Client() (*Client, error) {
	client := dnsimple.NewClient(dnsimple.NewOauthTokenCredentials(c.Token))
	client.UserAgent = "HashiCorp-Terraform/" + terraform.VersionString()

	provider := &Client{
		client: client,
		config: c,
	}

	log.Printf("[INFO] DNSimple Client configured for account: %s", c.Account)

	return provider, nil
}
