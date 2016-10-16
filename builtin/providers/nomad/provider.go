package nomad

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_ADDRESS", nil),
				Description: "The HTTP API address for a nomad server.",
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_REGION", nil),
				Description: "The region of the nomad server.",
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_USERNAME", ""),
				Description: "The username for auth with the nomad server.",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Required:    false,
				DefaultFunc: schema.EnvDefaultFunc("NOMAD_PASSWORD", ""),
				Description: "The password for auth with the nomad server.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"nomad_job": resourceNomadJob(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	username := ""
	if value, ok := d.Get("username").(string); ok {
		username = value
	}
	password := ""
	if value, ok := d.Get("password").(string); ok {
		password = value
	}

	config := Config{
		Address:  d.Get("address").(string),
		Region:   d.Get("region").(string),
		Username: username,
		Password: password,
	}

	return config.Client()
}
