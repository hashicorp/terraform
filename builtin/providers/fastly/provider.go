package fastly

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"FASTLY_API_KEY",
				}, nil),
				Description: "Fastly API Key from https://app.fastly.com/#account",
			},
		},
		DataSourcesMap: map[string]*schema.Resource{
			"fastly_ip_ranges": dataSourceFastlyIPRanges(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"fastly_service_v1": resourceServiceV1(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		ApiKey: d.Get("api_key").(string),
	}
	return config.Client()
}
