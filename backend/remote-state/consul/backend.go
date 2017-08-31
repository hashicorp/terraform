package consul

import (
	"context"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

// New creates a new backend for Consul remote state.
func New() backend.Backend {
	s := &schema.Backend{
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
				Default:     "", // To prevent input
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

			"gzip": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Compress the state data using gzip",
				Default:     false,
			},

			"lock": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Lock state access",
				Default:     true,
			},

			"ca_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CACERT", ""),
			},

			"cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CLIENT_CERT", ""),
			},

			"key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded private key, required if cert_file is specified.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CLIENT_KEY", ""),
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	// The fields below are set from configure
	configData *schema.ResourceData
	lock       bool
}

func (b *Backend) configure(ctx context.Context) error {
	// Grab the resource data
	b.configData = schema.FromContextBackendConfig(ctx)

	// Store the lock information
	b.lock = b.configData.Get("lock").(bool)

	// Initialize a client to test config
	_, err := b.clientRaw()
	return err
}

func (b *Backend) clientRaw() (*consulapi.Client, error) {
	data := b.configData

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

	tlsConfig := &consulapi.TLSConfig{}
	if v, ok := data.GetOk("ca_file"); ok && v.(string) != "" {
		tlsConfig.CAFile = v.(string)
	}
	if v, ok := data.GetOk("cert_file"); ok && v.(string) != "" {
		tlsConfig.CertFile = v.(string)
	}
	if v, ok := data.GetOk("key_file"); ok && v.(string) != "" {
		tlsConfig.KeyFile = v.(string)
	}
	cc, err := consulapi.SetupTLSConfig(tlsConfig)
	if err != nil {
		return nil, err
	}
	config.Transport.TLSClientConfig = cc

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

	return consulapi.NewClient(config)
}
