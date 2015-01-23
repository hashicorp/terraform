package main

import (
	"fmt"
	dme "github.com/soniah/dnsmadeeasy"
	"log"
)

// Config contains DNSMadeEasy provider settings
type Config struct {
	AKey       string
	SKey       string
	UseSandbox bool
}

// Client returns a new client for accessing DNSMadeEasy
func (c *Config) Client() (*dme.Client, error) {
	client, err := dme.NewClient(c.AKey, c.SKey)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	if c.UseSandbox {
		client.URL = dme.SandboxURL
	}

	log.Printf("[INFO] DNSMadeEasy Client configured for AKey: %s", client.AKey)

	return client, nil
}
