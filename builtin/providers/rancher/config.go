package rancher

import (
	"fmt"
	"log"
)

// Config - provider config
type Config struct {
	ServerUrl string
	AccessKey string
	SecretKey string
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
