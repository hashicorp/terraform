package consul

import (
	"context"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/backend/remote-state"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/state/remote"
)

// New creates a new backend for Consul remote state.
func New() backend.Backend {
	return &remotestate.Backend{
		ConfigureFunc: configure,

		// Set the schema
		Backend: &schema.Backend{
			Schema: map[string]*schema.Schema{
				"path": &schema.Schema{
					Type:        schema.TypeString,
					Required:    true,
					Description: "Path to store state in Consul",
				},

				"access_token": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Access token for a Consul ACL",
					Default:     "", // To prevent input
				},

				"address": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Address to the Consul Cluster",
				},

				"scheme": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Scheme to communicate to Consul with",
					Default:     "", // To prevent input
				},

				"datacenter": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Datacenter to communicate with",
					Default:     "", // To prevent input
				},

				"http_auth": &schema.Schema{
					Type:        schema.TypeString,
					Optional:    true,
					Description: "HTTP Auth in the format of 'username:password'",
					Default:     "", // To prevent input
				},
			},
		},
	}
}

func configure(ctx context.Context) (remote.Client, error) {
	// Grab the resource data
	data := schema.FromContextBackendConfig(ctx)

	// Configure the client
	config := consulapi.DefaultConfig()
	if v, ok := data.GetOk("access_token"); ok && v.(string) != "" {
		config.Token = v.(string)
	}
	if v, ok := data.GetOk("address"); ok && v.(string) != "" {
		config.Address = v.(string)
	}
	if v, ok := data.GetOk("scheme"); ok && v.(string) != "" {
		config.Scheme = v.(string)
	}
	if v, ok := data.GetOk("datacenter"); ok && v.(string) != "" {
		config.Datacenter = v.(string)
	}
	if v, ok := data.GetOk("http_auth"); ok && v.(string) != "" {
		auth := v.(string)

		var username, password string
		if strings.Contains(auth, ":") {
			split := strings.SplitN(auth, ":", 2)
			username = split[0]
			password = split[1]
		} else {
			username = auth
		}

		config.HttpAuth = &consulapi.HttpBasicAuth{
			Username: username,
			Password: password,
		}
	}

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &RemoteClient{
		Client: client,
		Path:   data.Get("path").(string),
	}, nil
}
