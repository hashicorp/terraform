package mailgun

import (
	"log"

	"github.com/pearkes/mailgun"
)

type Config struct {
	APIKey string
}

// Client() returns a new client for accessing mailgun.
//
func (c *Config) Client() (*mailgun.Client, error) {

	// We don't set a domain right away
	client, err := mailgun.NewClient(c.APIKey)

	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Mailgun Client configured ")

	return client, nil
}
