package pagerduty

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider represents a resource provider in Terraform
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PAGERDUTY_TOKEN", nil),
			},

			"skip_credentials_validation": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"pagerduty_user":              dataSourcePagerDutyUser(),
			"pagerduty_schedule":          dataSourcePagerDutySchedule(),
			"pagerduty_escalation_policy": dataSourcePagerDutyEscalationPolicy(),
			"pagerduty_vendor":            dataSourcePagerDutyVendor(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"pagerduty_addon":               resourcePagerDutyAddon(),
			"pagerduty_user":                resourcePagerDutyUser(),
			"pagerduty_team":                resourcePagerDutyTeam(),
			"pagerduty_service":             resourcePagerDutyService(),
			"pagerduty_service_integration": resourcePagerDutyServiceIntegration(),
			"pagerduty_schedule":            resourcePagerDutySchedule(),
			"pagerduty_escalation_policy":   resourcePagerDutyEscalationPolicy(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(data *schema.ResourceData) (interface{}, error) {
	config := Config{
		Token:               data.Get("token").(string),
		SkipCredsValidation: data.Get("skip_credentials_validation").(bool),
	}

	log.Println("[INFO] Initializing PagerDuty client")
	return config.Client()
}
