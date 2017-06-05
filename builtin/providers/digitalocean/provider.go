package digitalocean

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for DigitalOcean.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DIGITALOCEAN_TOKEN", nil),
				Description: "The token key for API operations.",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"digitalocean_image": dataSourceDigitalOceanImage(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"digitalocean_certificate":  resourceDigitalOceanCertificate(),
			"digitalocean_domain":       resourceDigitalOceanDomain(),
			"digitalocean_droplet":      resourceDigitalOceanDroplet(),
			"digitalocean_floating_ip":  resourceDigitalOceanFloatingIp(),
			"digitalocean_loadbalancer": resourceDigitalOceanLoadbalancer(),
			"digitalocean_record":       resourceDigitalOceanRecord(),
			"digitalocean_ssh_key":      resourceDigitalOceanSSHKey(),
			"digitalocean_tag":          resourceDigitalOceanTag(),
			"digitalocean_volume":       resourceDigitalOceanVolume(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Token: d.Get("token").(string),
	}

	return config.Client()
}
