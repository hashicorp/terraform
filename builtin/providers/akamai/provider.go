package akamai

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a schema.Provider for Akamai.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AKAMAI_EDGEGRID_HOST"),
			},
			"access_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AKAMAI_EDGEGRID_ACCESS_TOKEN"),
			},
			"client_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AKAMAI_EDGEGRID_CLIENT_TOKEN"),
			},
			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("AKAMAI_EDGEGRID_CLIENT_SECRET"),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"akamai_gtm_domain":      resourceAkamaiGTMDomain(),
			"akamai_gtm_property":    resourceAkamaiGTMProperty(),
			"akamai_gtm_data_center": resourceAkamaiGTMDataCenter(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}

		return nil, nil
	}
}

func envDefaultFuncAllowMissing(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		v := os.Getenv(k)
		return v, nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AccessToken:  d.Get("access_token").(string),
		ClientToken:  d.Get("client_token").(string),
		ClientSecret: d.Get("client_secret").(string),
		APIHost:      d.Get("host").(string),
	}

	return config.Client()
}
