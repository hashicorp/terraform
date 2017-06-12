package runscope

import (
	"github.com/ewilde/go-runscope"
	"log"
)

// Config contains runscope provider settings
type Config struct {
	AccessToken string
	ApiUrl      string
}

func (c *Config) Client() (*runscope.Client, error) {
	client := runscope.NewClient(c.ApiUrl, c.AccessToken)

	log.Printf("[INFO] runscope client configured for server %s", c.ApiUrl)

	return client, nil
}
