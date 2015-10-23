package datadog

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
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
			"datadog_dashboard":     resourceDatadogDashboard(),
			"datadog_graph":         resourceDatadogGraph(),
			"datadog_monitor":       resourceDatadogMonitor(),
			"datadog_service_check": resourceDatadogServiceCheck(),
			"datadog_metric_alert":  resourceDatadogMetricAlert(),
			"datadog_outlier_alert": resourceDatadogOutlierAlert(),
		},

		ConfigureFunc: providerConfigure,
	}
}

// ProviderConfigure returns a configured client.
func providerConfigure(d *schema.ResourceData) (interface{}, error) {

	config := Config{
		APIKey: d.Get("api_key").(string),
		APPKey: d.Get("app_key").(string),
	}

	log.Println("[INFO] Initializing Datadog client")
	return config.Client()
}
