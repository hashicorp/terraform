package namecheap

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_USERNAME", nil),
				Description: "A registered username for namecheap",
			},

			"api_user": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_API_USER", nil),
				Description: "A registered apiuser for namecheap",
			},

			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_TOKEN", nil),
				Description: "The token key for API operations.",
			},

			"ip": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_IP", nil),
				Description: "IP addess of the machine running terraform",
			},

			"use_sandbox": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_USE_SANDBOX", nil),
				Description: "If true, use the namecheap sandbox",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"namecheap_record": resourceNameCheapRecord(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		username:    d.Get("username").(string),
		api_user:    d.Get("api_user").(string),
		token:       d.Get("token").(string),
		ip:          d.Get("ip").(string),
		use_sandbox: d.Get("use_sandbox").(bool),
	}

	return config.Client()
}
