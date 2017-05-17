package profitbricks

import (
	"github.com/profitbricks/profitbricks-sdk-go"
)

type Config struct {
	Username string
	Password string
	Endpoint string
	Retries  int
}

// Client() returns a new client for accessing ProfitBricks.
func (c *Config) Client() (*Config, error) {
	profitbricks.SetAuth(c.Username, c.Password)
	profitbricks.SetDepth("5")
	if len(c.Endpoint) > 0 {
		profitbricks.SetEndpoint(c.Endpoint)
	}
	return c, nil
}
