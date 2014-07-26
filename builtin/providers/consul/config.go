package consul

import (
	"log"

	"github.com/armon/consul-api"
)

type Config struct {
	Datacenter string `mapstructure:"datacenter"`
	Address    string `mapstructure:"address"`
}

// Client() returns a new client for accessing digital
// ocean.
//
func (c *Config) Client() (*consulapi.Client, error) {
	config := consulapi.DefaultConfig()
	if c.Datacenter != "" {
		config.Datacenter = c.Datacenter
	}
	if c.Address != "" {
		config.Address = c.Address
	}
	client, err := consulapi.NewClient(config)

	log.Printf("[INFO] Consul Client configured with address: '%s', datacenter: '%s'",
		config.Address, config.Datacenter)
	if err != nil {
		return nil, err
	}
	return client, nil
}
