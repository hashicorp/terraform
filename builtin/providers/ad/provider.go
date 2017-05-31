package ad

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{

			"domain": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "domain",
				DefaultFunc: schema.EnvDefaultFunc("AD_DOMAIN", nil),
			},

			"ip": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ip",
				DefaultFunc: schema.EnvDefaultFunc("AD_IP", nil),
			},

			"user": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "user",
				DefaultFunc: schema.EnvDefaultFunc("AD_USER", nil),
			},

			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "password",
				DefaultFunc: schema.EnvDefaultFunc("AD_PASSWORD", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"ad_resourceComputer": resourceComputer(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	config := Config{
		Domain:   d.Get("domain").(string),
		IP:       d.Get("ip").(string),
		Username: d.Get("user").(string),
		Password: d.Get("password").(string),
	}
	log.Printf("[INFO] Connecting to AD")
	return config.Client()
}
