package newrelic

import (
	"log"

	"github.com/hashicorp/terraform/helper/logging"
	newrelic "github.com/paultyng/go-newrelic/api"
)

// Config contains New Relic provider settings
type Config struct {
	APIKey string
	APIURL string
}

// Client returns a new client for accessing New Relic
func (c *Config) Client() (*newrelic.Client, error) {
	nrConfig := newrelic.Config{
		APIKey:  c.APIKey,
		Debug:   logging.IsDebugOrHigher(),
		BaseURL: c.APIURL,
	}

	client := newrelic.New(nrConfig)

	log.Printf("[INFO] New Relic client configured")

	return &client, nil
}
