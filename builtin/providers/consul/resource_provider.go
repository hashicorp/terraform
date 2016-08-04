package consul

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"scheme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"tls": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"cert_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"key_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"consul_keys": dataSourceConsulKeys(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"consul_agent_service": resourceConsulAgentService(),
			"consul_catalog_entry": resourceConsulCatalogEntry(),
			"consul_keys":          resourceConsulKeys(),
			"consul_key_prefix":    resourceConsulKeyPrefix(),
			"consul_node":          resourceConsulNode(),
			"consul_service":       resourceConsulService(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var config Config
	configRaw := d.Get("").(map[string]interface{})
	if err := mapstructure.Decode(configRaw, &config); err != nil {
		return nil, err
	}
	log.Printf("[INFO] Initializing Consul client")
	return config.Client()
}
