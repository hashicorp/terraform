package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/profitbricks/profitbricks-sdk-go"
)

// Provider returns a schema.Provider for ProfitBricks.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROFITBRICKS_USERNAME", nil),
				Description: "ProfitBricks username for API operations.",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROFITBRICKS_PASSWORD", nil),
				Description: "ProfitBricks password for API operations.",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROFITBRICKS_API_URL", profitbricks.Endpoint),
				Description: "ProfitBricks REST API URL.",
			},
			"retries": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  50,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"profitbricks_datacenter":   resourceProfitBricksDatacenter(),
			"profitbricks_ipblock":      resourceProfitBricksIPBlock(),
			"profitbricks_firewall":     resourceProfitBricksFirewall(),
			"profitbricks_lan":          resourceProfitBricksLan(),
			"profitbricks_loadbalancer": resourceProfitBricksLoadbalancer(),
			"profitbricks_nic":          resourceProfitBricksNic(),
			"profitbricks_server":       resourceProfitBricksServer(),
			"profitbricks_volume":       resourceProfitBricksVolume(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"profitbricks_datacenter": dataSourceDataCenter(),
			"profitbricks_location":   dataSourceLocation(),
			"profitbricks_image":      dataSourceImage(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	if _, ok := d.GetOk("username"); !ok {
		return nil, fmt.Errorf("ProfitBricks username has not been provided.")
	}

	if _, ok := d.GetOk("password"); !ok {
		return nil, fmt.Errorf("ProfitBricks password has not been provided.")
	}

	config := Config{
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
		Endpoint: d.Get("endpoint").(string),
		Retries:  d.Get("retries").(int),
	}

	return config.Client()
}
