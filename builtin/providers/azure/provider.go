package azure

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"settings_file": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_SETTINGS_FILE", nil),
			},

			"subscription_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_SUBSCRIPTION_ID", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"azure_disk":     resourceAzureDisk(),
			"azure_instance": resourceAzureInstance(),
			"azure_network":  resourceAzureNetwork(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		SettingsFile:   d.Get("settings_file").(string),
		SubscriptionID: d.Get("subscription_id").(string),
	}

	return config.NewClient()
}
