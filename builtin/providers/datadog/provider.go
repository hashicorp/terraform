package datadog

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_API_KEY", nil),
			},
			"app_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_APP_KEY", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"datadog_monitor": resourceDatadogMonitor(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	config := Config{
		APIKey: d.Get("api_key").(string),
		APPKey: d.Get("app_key").(string),
	}

	log.Println("[INFO] Initializing Datadog client")
	return config.Client()
}
