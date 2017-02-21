package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
)

// Config defines the configuration options for the PagerDuty client
type Config struct {
	Token string
}

// Client returns a new PagerDuty client
func (c *Config) Client() (*pagerduty.Client, error) {
	client := pagerduty.NewClient(c.Token)

	log.Printf("[INFO] PagerDuty client configured")

	return client, nil
}
