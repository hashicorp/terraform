package vrealize

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Provider represents a Terraform provider
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The username for vRealize API operations.",
			},

			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The password for vRealize API operations.",
			},

			"tenant": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vRealize tenant name for vRealize API operations.",
			},

			"server": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The vRealize server name for vRealize API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"vrealize_machine": resourceMachine(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	config := Config{
		User:     d.Get("username").(string),
		Password: d.Get("password").(string),
		Tenant:   d.Get("tenant").(string),
		Server:   d.Get("server").(string),
	}

	return config.Client()
}
