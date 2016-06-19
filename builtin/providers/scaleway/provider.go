package scaleway

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SCALEWAY_API_KEY", nil),
				Description: "The API key for Scaleway API operations.",
			},
			"organization": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SCALEWAY_ORGANIZATION", nil),
				Description: "The Organization ID for Scaleway API operations.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"scaleway_server":              resourceScalewayServer(),
			"scaleway_ip":                  resourceScalewayIp(),
			"scaleway_security_group":      resourceScalewaySecurityGroup(),
			"scaleway_security_group_rule": resourceScalewaySecurityGroupRule(),
			"scaleway_volume":              resourceScalewayVolume(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		Organization: d.Get("organization").(string),
		ApiKey:       d.Get("api_key").(string),
	}

	return config.Client()
}

func String(str string) *string {
	return &str
}
