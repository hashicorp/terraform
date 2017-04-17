package consul

import (
	"log"
	"net/http"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

type Config struct {
	Datacenter string `mapstructure:"datacenter"`
	Address    string `mapstructure:"address"`
	Scheme     string `mapstructure:"scheme"`
	HttpAuth   string `mapstructure:"http_auth"`
	Token      string `mapstructure:"token"`
	CAFile     string `mapstructure:"ca_file"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
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

	tlsConfig := &consulapi.TLSConfig{}
	tlsConfig.CAFile = c.CAFile
	tlsConfig.CertFile = c.CertFile
	tlsConfig.KeyFile = c.KeyFile
	cc, err := consulapi.SetupTLSConfig(tlsConfig)
	if err != nil {
		return nil, err
	}
	config.HttpClient.Transport.(*http.Transport).TLSClientConfig = cc

	if c.HttpAuth != "" {
		var username, password string
		if strings.Contains(c.HttpAuth, ":") {
			split := strings.SplitN(c.HttpAuth, ":", 2)
			username = split[0]
			password = split[1]
		} else {
			username = c.HttpAuth
		}
		config.HttpAuth = &consulapi.HttpBasicAuth{Username: username, Password: password}
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
