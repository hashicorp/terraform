package opc

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/go-oracle-terraform/opc"
)

type Config struct {
	User            string
	Password        string
	IdentityDomain  string
	Endpoint        string
	MaxRetryTimeout int
}

type OPCClient struct {
	Client          *compute.Client
	MaxRetryTimeout int
}

func (c *Config) Client() (*compute.Client, error) {
	u, err := url.ParseRequestURI(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Invalid endpoint URI: %s", err)
	}

	config := opc.Config{
		IdentityDomain: &c.IdentityDomain,
		Username:       &c.User,
		Password:       &c.Password,
		APIEndpoint:    u,
		HTTPClient:     http.DefaultClient,
	}

	// TODO: http client wrapping / log level
	return compute.NewComputeClient(&config)
}
