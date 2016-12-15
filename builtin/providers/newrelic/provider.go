package newrelic

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a resource provider in Terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_key": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NEWRELIC_API_KEY", nil),
				Sensitive:   true,
			},
			"api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NEWRELIC_API_URL", "https://api.newrelic.com/v2"),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"newrelic_application": dataSourceNewRelicApplication(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"newrelic_alert_channel":        resourceNewRelicAlertChannel(),
			"newrelic_alert_condition":      resourceNewRelicAlertCondition(),
			"newrelic_alert_policy":         resourceNewRelicAlertPolicy(),
			"newrelic_alert_policy_channel": resourceNewRelicAlertPolicyChannel(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIKey: data.Get("api_key").(string),
		APIURL: data.Get("api_url").(string),
	}
	log.Println("[INFO] Initializing New Relic client")
	return config.Client()
}
