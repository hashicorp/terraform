package opsgenie

import (
	"log"

	"golang.org/x/net/context"

	"github.com/opsgenie/opsgenie-go-sdk/client"
)

type OpsGenieClient struct {
	apiKey string

	StopContext context.Context

	users client.OpsGenieUserClient
}

// Config defines the configuration options for the OpsGenie client
type Config struct {
	ApiKey string
}

// Client returns a new OpsGenie client
func (c *Config) Client() (*OpsGenieClient, error) {
	opsGenie := new(client.OpsGenieClient)
	opsGenie.SetAPIKey(c.ApiKey)

	log.Printf("[INFO] OpsGenie client configured")

	client := OpsGenieClient{}

	usersClient, err := opsGenie.User()
	if err != nil {
		return nil, err
	}
	client.users = *usersClient

	return &client, nil
}
