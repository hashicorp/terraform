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
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_SETTINGS_FILE", nil),
			},

			"subscription_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_SUBSCRIPTION_ID", ""),
			},

			"certificate": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_CERTIFICATE", ""),
			},

			"management_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("AZURE_MANAGEMENT_URL", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"azure_data_disk":       resourceAzureDataDisk(),
			"azure_instance":        resourceAzureInstance(),
			"azure_security_group":  resourceAzureSecurityGroup(),
			"azure_virtual_network": resourceAzureVirtualNetwork(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		SettingsFile:   d.Get("settings_file").(string),
		SubscriptionID: d.Get("subscription_id").(string),
		Certificate:    []byte(d.Get("certificate").(string)),
		ManagementURL:  d.Get("management_url").(string),
	}

	return config.NewClient()
}
