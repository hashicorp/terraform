package heroku

import (
	"log"
	"os"

	"github.com/bgentry/heroku-go"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	Email  string `mapstructure:"email"`
}

// Client() returns a new client for accessing heroku.
//
func (c *Config) Client() (*heroku.Client, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("HEROKU_EMAIL"); v != "" {
		c.Email = v
	}
	if v := os.Getenv("HEROKU_API_KEY"); v != "" {
		c.APIKey = v
	}

	client := heroku.Client{Username: c.Email, Password: c.APIKey}

	log.Printf("[INFO] Heroku Client configured for user: %s", c.Email)

	return &client, nil
}
