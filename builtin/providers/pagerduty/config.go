package pagerduty

import (
	"fmt"
	"log"

	"github.com/PagerDuty/go-pagerduty"
)

// Config defines the configuration options for the PagerDuty client
type Config struct {
	Token string
}

const invalidCredentials = `

No valid credentials found for PagerDuty provider.
Please see https://www.terraform.io/docs/providers/pagerduty/index.html
for more information on providing credentials for this provider.
`

// Client returns a new PagerDuty client
func (c *Config) Client() (*pagerduty.Client, error) {
	// Validate that the PagerDuty token is set
	if c.Token == "" {
		return nil, fmt.Errorf(invalidCredentials)
	}

	client := pagerduty.NewClient(c.Token)

	// Validate the credentials by calling the abilities endpoint,
	// if we get a 401 response back we return an error to the user
	if _, err := client.ListAbilities(); err != nil {
		if isUnauthorized(err) {
			return nil, fmt.Errorf(invalidCredentials)
		}
		return nil, err
	}

	log.Printf("[INFO] PagerDuty client configured")

	return client, nil
}
