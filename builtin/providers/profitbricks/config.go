package profitbricks

import (
	"github.com/profitbricks/profitbricks-sdk-go"
)

type Config struct {
	Username string
	Password string
	Retries  int
}

// Client() returns a new client for accessing digital ocean.
func (c *Config) Client() (*Config, error) {
	profitbricks.SetAuth(c.Username, c.Password)
	profitbricks.SetDepth("5")

	return c, nil
}
