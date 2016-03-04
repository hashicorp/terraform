package powerdns

import (
	"fmt"
	"log"
)

type Config struct {
	ServerUrl string
	ApiKey    string
}

// Client returns a new client for accessing PowerDNS
func (c *Config) Client() (*Client, error) {
	client, err := NewClient(c.ServerUrl, c.ApiKey)

	if err != nil {
		return nil, fmt.Errorf("Error setting up PowerDNS client: %s", err)
	}

	log.Printf("[INFO] PowerDNS Client configured for server %s", c.ServerUrl)

	return client, nil
}
