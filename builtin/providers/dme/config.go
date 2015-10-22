package dme

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/soniah/dnsmadeeasy"
)

// Config contains DNSMadeEasy provider settings
type Config struct {
	AKey       string
	SKey       string
	UseSandbox bool
}

// Client returns a new client for accessing DNSMadeEasy
func (c *Config) Client() (*dnsmadeeasy.Client, error) {
	client, err := dnsmadeeasy.NewClient(c.AKey, c.SKey)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	client.HTTP = cleanhttp.DefaultClient()

	if c.UseSandbox {
		client.URL = dnsmadeeasy.SandboxURL
	}

	log.Printf("[INFO] DNSMadeEasy Client configured for AKey: %s", client.AKey)

	return client, nil
}
