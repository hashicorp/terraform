package opsgenie

import (
	"log"
	"github.com/opsgenie/opsgenie-go-sdk/client"
)

// Config defines the configuration options for the OpsGenie client
type Config struct {
	ApiKey string
}

// Client returns a new OpsGenie client
func (c *Config) Client() (*client.OpsGenieClient, error) {
	opsGenie := new(client.OpsGenieClient)
	opsGenie.SetAPIKey(c.ApiKey)

	log.Printf("[INFO] OpsGenie client configured")

	return opsGenie, nil
}
