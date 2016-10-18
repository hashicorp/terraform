package maas

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"log"
)

func Provider() terraform.ResourceProvider {
    log.Println("[DEBUG] Initializing the MAAS provider")
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The api key for API operations",
			},
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The MAAS server URL. ie: http://1.2.3.4:80/MAAS",
			},
			"api_version": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The MAAS API version. Currently: 1.0",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"maas_instance": resourceMAASInstance(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
    log.Println("[DEBUG] Configuring the MAAS provider")
	config := Config{
		APIKey: d.Get("api_key").(string),
		APIURL: d.Get("api_url").(string),
		APIver: d.Get("api_version").(string),
	}
	return config.Client()
}
