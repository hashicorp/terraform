package consul

import (
	"log"

	consulapi "github.com/hashicorp/consul/api"
)

type Config struct {
	Datacenter string `mapstructure:"datacenter"`
	Address    string `mapstructure:"address"`
	Scheme     string `mapstructure:"scheme"`
}

// Client() returns a new client for accessing consul.
//
func (c *Config) Client() (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	if c.Datacenter != "" {
		config.Datacenter = c.Datacenter
	}
	if c.Address != "" {
		config.Address = c.Address
	}
	if c.Scheme != "" {
		config.Scheme = c.Scheme
	}
	client, err := consulapi.NewClient(config)

	log.Printf("[INFO] Consul Client configured with address: '%s', scheme: '%s', datacenter: '%s'",
		config.Address, config.Scheme, config.Datacenter)
	if err != nil {
		return nil, err
	}
	return client, nil
}
