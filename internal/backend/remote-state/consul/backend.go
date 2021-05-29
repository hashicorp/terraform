package consul

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	vaultapi "github.com/hashicorp/vault/api"
)

// New creates a new backend for Consul remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to store state in Consul",
			},

			"access_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Access token for a Consul ACL",
				Default:     "", // To prevent input
			},

			"address": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Address to the Consul Cluster",
				Default:     "", // To prevent input
			},

			"scheme": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Scheme to communicate to Consul with",
				Default:     "", // To prevent input
			},

			"datacenter": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Datacenter to communicate with",
				Default:     "", // To prevent input
			},

			"http_auth": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "HTTP Auth in the format of 'username:password'",
				Default:     "", // To prevent input
			},

			"gzip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Compress the state data using gzip",
				Default:     false,
			},

			"lock": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Lock state access",
				Default:     true,
			},

			"ca_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded certificate authority used to verify the remote agent's certificate.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CACERT", ""),
			},

			"cert_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CLIENT_CERT", ""),
			},

			"key_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded private key, required if cert_file is specified.",
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CLIENT_KEY", ""),
			},

			"vault": {
				Type: schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Type: schema.TypeString,
							Required: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_ADDR", nil),
						},
						"token": {
							Type: schema.TypeString,
							Required: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_TOKEN", nil),
						},
						"key_name": {
							Type: schema.TypeString,
							Required: true,
							DefaultFunc: schema.EnvDefaultFunc("TRANSIT_KEY_NAME", nil),
						},
						"context": {
							Type: schema.TypeString,
							Optional: true,
						},
						"mount_path": {
							Type: schema.TypeString,
							Optional: true,
							Default: "transit/",
						},
						"namespace": {
							Type:schema.TypeString,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_NAMESPACE", nil),
						},
						"tls_ca_cert": {
							Type: schema.TypeString,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_CA_CERT", nil),
						},
						"tls_client_cert": {
							Type: schema.TypeString,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_CLIENT_CERT", nil),
						},
						"tls_client_key": {
							Type: schema.TypeString,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_CLIENT_KEY", nil),
						},
						"tls_server_name": {
							Type: schema.TypeString,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_TLS_SERVER_NAME", nil),
						},
						"tls_skip_verify": {
							Type: schema.TypeBool,
							Optional: true,
							DefaultFunc: schema.EnvDefaultFunc("VAULT_SKIP_VERIFY", nil),
						},
					},
				},
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
	client     *consulapi.Client
	configData *schema.ResourceData
	lock       bool
	transit *TransitClient
}

func (b *Backend) configure(ctx context.Context) error {
	// Grab the resource data
	b.configData = schema.FromContextBackendConfig(ctx)

	// Store the lock information
	b.lock = b.configData.Get("lock").(bool)

	data := b.configData

	// Configure the client
	config := consulapi.DefaultConfig()

	// replace the default Transport Dialer to reduce the KeepAlive
	config.Transport.DialContext = dialContext

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

	if v, ok := data.GetOk("ca_file"); ok && v.(string) != "" {
		config.TLSConfig.CAFile = v.(string)
	}
	if v, ok := data.GetOk("cert_file"); ok && v.(string) != "" {
		config.TLSConfig.CertFile = v.(string)
	}
	if v, ok := data.GetOk("key_file"); ok && v.(string) != "" {
		config.TLSConfig.KeyFile = v.(string)
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
		return err
	}

	b.client = client

	vault := b.configData.Get("vault").([]interface{})
	if len(vault) == 0 {
		return nil
	}

	vData := vault[0].(map[string]interface{})

	vConfig := vaultapi.DefaultConfig()
	if vConfig == nil {
		return fmt.Errorf("failed to configure Vault client: %s", err)
	}
	if addr := vData["address"].(string); addr != "" {
		vConfig.Address = addr
	}

	tls := &vaultapi.TLSConfig{
		CACert: vData["tls_ca_cert"].(string),
		ClientCert: vData["tls_client_cert"].(string),
		ClientKey: vData["tls_client_key"].(string),
		TLSServerName: vData["tls_server_name"].(string),
		Insecure: vData["tls_skip_verify"].(bool),
	}
	if err := vConfig.ConfigureTLS(tls); err != nil {
		return fmt.Errorf("failed to configure Vault client: %s", err)
	}

	vaultClient, err := vaultapi.NewClient(vConfig)
	if err != nil {
		return fmt.Errorf("failed to get Vault client: %s", err)
	}

	if token := vData["token"].(string); token != "" {
		vaultClient.SetToken(token)
	}
	if namespace := vData["namespace"].(string); namespace != "" {
		vaultClient.SetNamespace(namespace)
	}
	b.transit = &TransitClient{
		client: vaultClient,
		mountPath: vData["mount_path"].(string),
		keyName: vData["key_name"].(string),
		context: vData["context"].(string),
	}

	return nil
}

// dialContext is the DialContext function for the consul client transport.
// This is stored in a package var to inject a different dialer for tests.
var dialContext = (&net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 17 * time.Second,
}).DialContext
