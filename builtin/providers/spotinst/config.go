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
	msg := "%s\n\nNo valid credentials found for Spotinst Provider.\nPlease see https://www.terraform.io/docs/providers/spotinst/index.html\nfor more information on providing credentials for Spotinst Provider."

	if c.Password != "" && c.Token != "" {
		err := "ERR_CONFLICT: Both a password and a token were set, only one is required"
		return fmt.Errorf(msg, err)
	}

	if c.Password != "" && (c.Email == "" || c.ClientID == "" || c.ClientSecret == "") {
		err := "ERR_MISSING: A password was set without email, client_id or client_secret"
		return fmt.Errorf(msg, err)
	}

	if c.Password == "" && c.Token == "" {
		err := "ERR_MISSING: A token is required if not using password"
		return fmt.Errorf(msg, err)
	}

	return nil
}

// Client returns a new client for accessing Spotinst.
func (c *Config) Client() (*spotinst.Client, error) {
	var clientOpts []spotinst.ClientOptionFunc
	if c.Token != "" {
		clientOpts = append(clientOpts, spotinst.SetToken(c.Token))
	} else {
		clientOpts = append(clientOpts, spotinst.SetCredentials(c.Email, c.Password, c.ClientID, c.ClientSecret))
	}
	client, err := spotinst.NewClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}
	log.Printf("[INFO] Spotinst client configured")
	return client, nil
}
