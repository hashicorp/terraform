package opc

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/go-oracle-terraform/opc"
	"github.com/hashicorp/terraform/helper/logging"
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
		HTTPClient:     cleanhttp.DefaultClient(),
	}

	if logging.IsDebugOrHigher() {
		config.LogLevel = opc.LogDebug
		config.Logger = opcLogger{}
	}

	return compute.NewComputeClient(&config)
}

type opcLogger struct{}

func (l opcLogger) Log(args ...interface{}) {
	tokens := make([]string, 0, len(args))
	for _, arg := range args {
		if token, ok := arg.(string); ok {
			tokens = append(tokens, token)
		}
	}
	log.Printf("[DEBUG] [go-oracle-terraform]: %s", strings.Join(tokens, " "))
}
