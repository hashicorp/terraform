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

			"apiuser": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_APIUSER", nil),
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

			"usesandbox": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NAMECHEAP_USESANDBOX", nil),
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
		UserName:   d.Get("username").(string),
		ApiUser:    d.Get("apiuser").(string),
		Token:      d.Get("token").(string),
		Ip:         d.Get("ip").(string),
		UseSandbox: d.Get("usesandbox").(bool),
	}

	return config.Client()
}
