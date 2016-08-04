package consul

import (
	"log"
	"net/http"

	consulapi "github.com/hashicorp/consul/api"
)

type Config struct {
	Datacenter string     `mapstructure:"datacenter"`
	Address    string     `mapstructure:"address"`
	Scheme     string     `mapstructure:"scheme"`
	TLS        *TLSConfig `mapstructure:"tls"`
	Token      string     `mapstructure:"token"`
}

type TLSConfig struct {
	CAFile   string `mapstructure:"ca_file"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
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

	if c.TLS != nil {
		tlsConfig := &consulapi.TLSConfig{}
		if c.TLS.CAFile != "" {
			tlsConfig.CAFile = c.TLS.CAFile
		}
		if c.TLS.CertFile != "" {
			tlsConfig.CertFile = c.TLS.CertFile
		}
		if c.TLS.KeyFile != "" {
			tlsConfig.KeyFile = c.TLS.KeyFile
		}
		cc, err := consulapi.SetupTLSConfig(tlsConfig)
		if err != nil {
			return nil, err
		}
		config.HttpClient.Transport.(*http.Transport).TLSClientConfig = cc
	}

	if c.Token != "" {
		config.Token = c.Token
	}

	client, err := consulapi.NewClient(config)

	log.Printf("[INFO] Consul Client configured with address: '%s', scheme: '%s', datacenter: '%s'",
		config.Address, config.Scheme, config.Datacenter)
	if err != nil {
		return nil, err
	}
	return client, nil
}
