package mailgun

import (
	"log"
	"os"

	"github.com/pearkes/mailgun"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
}

// Client() returns a new client for accessing mailgun.
//
func (c *Config) Client() (*mailgun.Client, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("MAILGUN_API_KEY"); v != "" {
		c.APIKey = v
	}

	// We don't set a domain right away
	client, err := mailgun.NewClient(c.APIKey)

	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] Mailgun Client configured ")

	return client, nil
}
