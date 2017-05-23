package oneandone

import (
	"github.com/1and1/oneandone-cloudserver-sdk-go"
)

type Config struct {
	Token    string
	Retries  int
	Endpoint string
	API      *oneandone.API
}

func (c *Config) Client() (*Config, error) {
	token := oneandone.SetToken(c.Token)

	if len(c.Endpoint) > 0 {
		c.API = oneandone.New(token, c.Endpoint)
	} else {
		c.API = oneandone.New(token, oneandone.BaseUrl)
	}

	return c, nil
}
