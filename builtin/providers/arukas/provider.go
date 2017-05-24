package arukas

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(JSONTokenParamName, nil),
				Description: "your Arukas APIKey(token)",
			},
			"secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc(JSONSecretParamName, nil),
				Description: "your Arukas APIKey(secret)",
			},
			"api_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(JSONUrlParamName, "https://app.arukas.io/api/"),
				Description: "default Arukas API url",
			},
			"trace": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(JSONDebugParamName, ""),
			},
			"timeout": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc(JSONTimeoutParamName, "900"),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"arukas_container": resourceArukasContainer(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	config := Config{
		Token:   d.Get("token").(string),
		Secret:  d.Get("secret").(string),
		URL:     d.Get("api_url").(string),
		Trace:   d.Get("trace").(string),
		Timeout: d.Get("timeout").(int),
	}

	return config.NewClient()
}
