package dyn

import (
	"fmt"
	"log"

	"github.com/nesv/go-dynect/dynect"
)

type Config struct {
	CustomerName string
	Username     string
	Password     string
}

// Client() returns a new client for accessing dyn.
func (c *Config) Client() (*dynect.ConvenientClient, error) {
	client := dynect.NewConvenientClient(c.CustomerName)
	err := client.Login(c.Username, c.Password)

	if err != nil {
		return nil, fmt.Errorf("Error setting up Dyn client: %s", err)
	}

	log.Printf("[INFO] Dyn client configured for customer: %s, user: %s", c.CustomerName, c.Username)

	return client, nil
}
