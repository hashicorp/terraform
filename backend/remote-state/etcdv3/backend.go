package etcd

import (
	"context"

	etcdv3 "github.com/coreos/etcd/clientv3"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
)

func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"endpoints": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				MinItems:    1,
				Required:    true,
				Description: "Endpoints for the etcd cluster.",
			},

			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Username used to connect to the etcd cluster.",
				Default:     "", // To prevent input.
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Password used to connect to the etcd cluster.",
				Default:     "", // To prevent input.
			},

			"prefix": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "An optional prefix to be added to keys when to storing state in etcd.",
				Default:     "",
			},

			"lock": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to lock state access.",
				Default:     true,
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
	data   *schema.ResourceData
	lock   bool
	prefix string
}

func (b *Backend) configure(ctx context.Context) error {
	// Grab the resource data.
	b.data = schema.FromContextBackendConfig(ctx)
	// Store the lock information.
	b.lock = b.data.Get("lock").(bool)
	// Store the prefix information.
	b.prefix = b.data.Get("prefix").(string)
	// Initialize a client to test config.
	_, err := b.rawClient()
	// Return err, if any.
	return err
}

func (b *Backend) rawClient() (*etcdv3.Client, error) {
	config := etcdv3.Config{}

	if v, ok := b.data.GetOk("endpoints"); ok {
		config.Endpoints = retrieveEndpoints(v)
	}
	if v, ok := b.data.GetOk("username"); ok && v.(string) != "" {
		config.Username = v.(string)
	}
	if v, ok := b.data.GetOk("password"); ok && v.(string) != "" {
		config.Password = v.(string)
	}

	return etcdv3.New(config)
}

func retrieveEndpoints(v interface{}) []string {
	var endpoints []string
	list := v.([]interface{})
	for _, ep := range list {
		endpoints = append(endpoints, ep.(string))
	}
	return endpoints
}
