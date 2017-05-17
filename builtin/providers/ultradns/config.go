package ultradns

import (
	"fmt"
	"log"

	"github.com/Ensighten/udnssdk"
)

// Config collects the connection service-endpoint and credentials
type Config struct {
	Username string
	Password string
	BaseURL  string
}

// Client returns a new client for accessing UltraDNS.
func (c *Config) Client() (*udnssdk.Client, error) {
	client, err := udnssdk.NewClient(c.Username, c.Password, c.BaseURL)

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] UltraDNS Client configured for user: %s", c.Username)

	return client, nil
}
