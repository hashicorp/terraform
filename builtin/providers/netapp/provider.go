package netapp

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a resource provider in Terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_EMAIL", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_PASSWORD", nil),
				Sensitive:   true,
			},
			"host": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("NETAPP_HOST", nil),
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"netapp_cloud_workenv": dataSourceWorkingEnvironments(),
		},

		// ResourcesMap: map[string]*schema.Resource{
		//   "newrelic_alert_channel":        resourceNewRelicAlertChannel(),
		//   "newrelic_alert_condition":      resourceNewRelicAlertCondition(),
		//   "newrelic_alert_policy":         resourceNewRelicAlertPolicy(),
		//   "newrelic_alert_policy_channel": resourceNewRelicAlertPolicyChannel(),
		// },

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		Host:     data.Get("host").(string),
		Email:    data.Get("email").(string),
		Password: data.Get("password").(string),
	}

	apis, err := config.APIs()
	if err != nil {
		return nil, fmt.Errorf("Error creating APIs: %s", err)
	}

	log.Println("[INFO] Initializing NetApp client")

	err = apis.AuthAPI.Login(config.Email, config.Password)
	if err != nil {
		return nil, fmt.Errorf("Error logging in user %s: %s", config.Email, err)
	}

	return apis, nil
}
