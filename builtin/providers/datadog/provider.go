package datadog

import (
	"log"

	"errors"

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
			"datadog_downtime":  resourceDatadogDowntime(),
			"datadog_monitor":   resourceDatadogMonitor(),
			"datadog_timeboard": resourceDatadogTimeboard(),
			"datadog_user":      resourceDatadogUser(),
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
	client := config.Client()

	ok, err := client.Validate()

	if err != nil {
		return client, err
	}

	if ok == false {
		return client, errors.New(`No valid credential sources found for Datadog Provider. Please see https://terraform.io/docs/providers/datadog/index.html for more information on providing credentials for the Datadog Provider`)
	}

	return client, nil
}
