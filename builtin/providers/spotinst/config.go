package spotinst

import (
	"fmt"
	"log"

	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

type Config struct {
	Email        string
	Password     string
	ClientID     string
	ClientSecret string
	Token        string
}

// Validate returns an error in case of invalid configuration.
func (c *Config) Validate() error {
	if c.Password != "" && c.Token != "" {
		return fmt.Errorf("ERR_CONFLICT: Both a password and a token were set, only one is required")
	}

	if c.Password != "" && (c.Email == "" || c.ClientID == "" || c.ClientSecret == "") {
		return fmt.Errorf("ERR_MISSING: A password was set without email, client_id or client_secret")
	}

	if c.Password == "" && c.Token == "" {
		return fmt.Errorf("ERR_MISSING: A token is required if not using password")
	}

	return nil
}

// Client returns a new client for accessing Spotinst.
func (c *Config) Client() (*spotinst.Client, error) {
	client, err := spotinst.NewClient(&spotinst.Credentials{
		Email:        c.Email,
		Password:     c.Password,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Token:        c.Token,
	})

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	log.Printf("[INFO] Spotinst client configured")

	return client, nil
}
