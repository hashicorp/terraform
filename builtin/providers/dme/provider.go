package dme

import (
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider provides a Provider...
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"akey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("DME_AKEY"),
				Description: "A DNSMadeEasy API Key.",
			},
			"skey": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: envDefaultFunc("DME_SKEY"),
				Description: "The Secret Key for API operations.",
			},
			"usesandbox": &schema.Schema{
				Type:        schema.TypeBool,
				Required:    true,
				DefaultFunc: envDefaultFunc("DME_USESANDBOX"),
				Description: "If true, use the DME Sandbox.",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"dme_record": resourceDMERecord(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func envDefaultFunc(k string) schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		if v := os.Getenv(k); v != "" {
			if v == "true" {
				return true, nil
			} else if v == "false" {
				return false, nil
			}
			return v, nil
		}
		return nil, nil
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		AKey:       d.Get("akey").(string),
		SKey:       d.Get("skey").(string),
		UseSandbox: d.Get("usesandbox").(bool),
	}
	return config.Client()
}
