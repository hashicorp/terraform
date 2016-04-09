package powerdns

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PDNS_API_KEY", nil),
				Description: "REST API authentication key",
			},
			"server_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PDNS_SERVER_URL", nil),
				Description: "Location of PowerDNS server",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"powerdns_record": resourcePDNSRecord(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		ApiKey:    data.Get("api_key").(string),
		ServerUrl: data.Get("server_url").(string),
	}

	return config.Client()
}
