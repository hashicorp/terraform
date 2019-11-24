package tikv

import (
	"context"

	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"
	"github.com/tikv/client-go/txnkv"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"pd_address": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems:    1,
				Required:    true,
				Description: "address of the tikv pd cluster.",
			},

			"prefix": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to store state in TiKV",
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
			},

			"cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded certificate provided to the remote agent; requires use of key_file.",
			},

			"key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A path to a PEM-encoded private key, required if cert_file is specified.",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	// The fields below are set from configure.
	rawKvClient *rawkv.Client
	txnKvClient *txnkv.Client
	data        *schema.ResourceData
	lock        bool
}

func (b *Backend) configure(ctx context.Context) error {
	var err error

	// Grab the resource data.
	b.data = schema.FromContextBackendConfig(ctx)

	// Store the lock information.
	b.lock = b.data.Get("lock").(bool)
	cfg := config.Default()
	if v, ok := b.data.GetOk("ca_file"); ok && v.(string) != "" {
		cfg.RPC.Security.SSLCA = v.(string)
	}
	if v, ok := b.data.GetOk("cert_file"); ok && v.(string) != "" {
		cfg.RPC.Security.SSLCert = v.(string)
	}
	if v, ok := b.data.GetOk("key_file"); ok && v.(string) != "" {
		cfg.RPC.Security.SSLKey = v.(string)
	}

	// Initialize tikv client
	pdAddresses := retrieveAddresses(b.data.Get("pd_address"))
	b.rawKvClient, err = rawkv.NewClient(ctx, pdAddresses, cfg)
	if err != nil {
		return err
	}

	b.txnKvClient, err = txnkv.NewClient(ctx, pdAddresses, cfg)
	if err != nil {
		return err
	}

	return err
}

func retrieveAddresses(v interface{}) []string {
	var addresses []string
	list := v.([]interface{})
	for _, addr := range list {
		addresses = append(addresses, addr.(string))
	}
	return addresses
}
