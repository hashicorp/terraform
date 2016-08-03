package profitbricks

import (
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"strconv"
)

type Config struct {
	Username string
	Password string
	Timeout  int
}

// Client() returns a new client for accessing digital ocean.
func (c *Config) Client() (string, error) {
	profitbricks.SetAuth(c.Username, c.Password)
	profitbricks.SetDepth("5")
	log.Printf("[DEBUG] Username and password %s : %s", c.Username, c.Password)
	log.Printf("[DEBUG] Timeout %d", c.Timeout)
	return c.Username + "," + c.Password + "," + strconv.Itoa(c.Timeout), nil
}
