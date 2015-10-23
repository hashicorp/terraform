package datadog

import (
	"log"

	"github.com/zorkian/go-datadog-api"
)

// Config holds API and APP keys to authenticate to Datadog.
type Config struct {
	APIKey string
	APPKey string
}

// Client returns a new Datadog client.
func (c *Config) Client() (*datadog.Client, error) {

	client := datadog.NewClient(c.APIKey, c.APPKey)

	log.Printf("[INFO] Datadog Client configured ")

	return client, nil
}
