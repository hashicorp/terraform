package digitalocean

import (
	"log"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

type Config struct {
	Token string
}

// Client() returns a new client for accessing digital ocean.
func (c *Config) Client() (*godo.Client, error) {
	tokenSrc := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: c.Token,
	})

	client := godo.NewClient(oauth2.NewClient(oauth2.NoContext, tokenSrc))

	log.Printf("[INFO] DigitalOcean Client configured for URL: %s", client.BaseURL.String())

	return client, nil
}
