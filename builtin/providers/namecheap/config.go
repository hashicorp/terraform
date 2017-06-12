package namecheap

import (
	"fmt"
	"github.com/HX-Rd/namecheap"
)

type Config struct {
	username    string
	api_user    string
	token       string
	ip          string
	use_sandbox bool
}

// Client() returns a new client for accessing Namecheap.
func (c *Config) Client() (*namecheap.Client, error) {
	client, err := namecheap.NewClient(c.username, c.api_user, c.token, c.ip, c.use_sandbox)

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	return client, nil
}
