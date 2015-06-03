package datadog

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
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_API_KEY", nil),
			},
			"app_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("DATADOG_APP_KEY", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"datadog_monitor_metric": datadogMonitorResource(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(rd *schema.ResourceData) (interface{}, error) {
	apiKey := rd.Get("api_key").(string)
	appKey := rd.Get("app_key").(string)
	return map[string]string{"api_key": apiKey, "app_key": appKey}, nil
}
