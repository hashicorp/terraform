package heroku

import (
	"log"
	"net/http"
	"os"

	"github.com/cyberdelia/heroku-go/v3"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	Email  string `mapstructure:"email"`
}

// Client() returns a new Service for accessing Heroku.
//
func (c *Config) Client() (*heroku.Service, error) {

	// If we have env vars set (like in the acc) tests,
	// we need to override the values passed in here.
	if v := os.Getenv("HEROKU_EMAIL"); v != "" {
		c.Email = v
	}
	if v := os.Getenv("HEROKU_API_KEY"); v != "" {
		c.APIKey = v
	}

	service := heroku.NewService(&http.Client{
		Transport: &heroku.Transport{
			Username:  c.Email,
			Password:  c.APIKey,
			UserAgent: heroku.DefaultUserAgent,
		},
	})

	log.Printf("[INFO] Heroku Client configured for user: %s", c.Email)

	return service, nil
}
