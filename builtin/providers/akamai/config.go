package akamai

import (
	"log"

	"github.com/Comcast/go-edgegrid/edgegrid"
)

// Config is the configuration required to instantiate
// Akamai API clients
type Config struct {
	AccessToken  string
	ClientToken  string
	ClientSecret string
	APIHost      string
}

// Clients contains Akamai GTM and PAPI clients for
// accessing the Akamai API.
type Clients struct {
	GTM  *edgegrid.GTMClient
	PAPI *edgegrid.PAPIClient
}

// Client returns a new AkamaiClients for accessing Akamai.
func (c *Config) Client() (*Clients, error) {
	clients := &Clients{
		edgegrid.GTMClientWithCreds(c.AccessToken, c.ClientToken, c.ClientSecret, c.APIHost),
		edgegrid.PAPIClientWithCreds(c.AccessToken, c.ClientToken, c.ClientSecret, c.APIHost),
	}

	log.Printf("[INFO] Akamai GTM and PAPI API Clients configured for use")

	return clients, nil
}
