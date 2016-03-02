package namecheap

import (
	"fmt"
	"github.com/HX-Rd/namecheap"
)

type Config struct {
	UserName   string
	ApiUser    string
	Token      string
	Ip         string
	UseSandbox bool
}

// Client() returns a new client for accessing namne cheap.
func (c *Config) Client() (*namecheap.Client, error) {
	client, err := namecheap.NewClient(c.UserName, c.ApiUser, c.Token, c.Ip, c.UseSandbox)

	if err != nil {
		return nil, fmt.Errorf("Error setting up client: %s", err)
	}

	return client, nil
}
