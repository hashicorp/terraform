package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for DigitalOcean.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROFITBRICKS_USERNAME", nil),
				Description: "Profitbricks username for API operations.",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PROFITBRICKS_PASSWORD", nil),
				Description: "Profitbricks password for API operations.",
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
		Retries:  d.Get("retries").(int),
	}

	return config.Client()
}
