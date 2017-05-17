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
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"CONSUL_ADDRESS",
					"CONSUL_HTTP_ADDR",
				}, "localhost:8500"),
			},

			"scheme": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"CONSUL_SCHEME",
					"CONSUL_HTTP_SCHEME",
				}, "http"),
			},

			"http_auth": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_HTTP_AUTH", ""),
			},

			"ca_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CA_FILE", ""),
			},

			"cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_CERT_FILE", ""),
			},

			"key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONSUL_KEY_FILE", ""),
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"CONSUL_TOKEN",
					"CONSUL_HTTP_TOKEN",
				}, ""),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"consul_agent_self":       dataSourceConsulAgentSelf(),
			"consul_catalog_nodes":    dataSourceConsulCatalogNodes(),
			"consul_catalog_service":  dataSourceConsulCatalogService(),
			"consul_catalog_services": dataSourceConsulCatalogServices(),
			"consul_keys":             dataSourceConsulKeys(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"consul_agent_service":  resourceConsulAgentService(),
			"consul_catalog_entry":  resourceConsulCatalogEntry(),
			"consul_keys":           resourceConsulKeys(),
			"consul_key_prefix":     resourceConsulKeyPrefix(),
			"consul_node":           resourceConsulNode(),
			"consul_prepared_query": resourceConsulPreparedQuery(),
			"consul_service":        resourceConsulService(),
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
