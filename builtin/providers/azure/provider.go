package azure

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
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
		},

		ResourcesMap: map[string]*schema.Resource{
			"azure_instance":                          resourceAzureInstance(),
			"azure_affinity_group":                    resourceAzureAffinityGroup(),
			"azure_data_disk":                         resourceAzureDataDisk(),
			"azure_sql_database_server":               resourceAzureSqlDatabaseServer(),
			"azure_sql_database_server_firewall_rule": resourceAzureSqlDatabaseServerFirewallRule(),
			"azure_sql_database_service":              resourceAzureSqlDatabaseService(),
			"azure_hosted_service":                    resourceAzureHostedService(),
			"azure_storage_service":                   resourceAzureStorageService(),
			"azure_storage_container":                 resourceAzureStorageContainer(),
			"azure_storage_blob":                      resourceAzureStorageBlob(),
			"azure_storage_queue":                     resourceAzureStorageQueue(),
			"azure_virtual_network":                   resourceAzureVirtualNetwork(),
			"azure_dns_server":                        resourceAzureDnsServer(),
			"azure_local_network_connection":          resourceAzureLocalNetworkConnection(),
			"azure_security_group":                    resourceAzureSecurityGroup(),
			"azure_security_group_rule":               resourceAzureSecurityGroupRule(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	settingsFile, err := homedir.Expand(d.Get("settings_file").(string))
	if err != nil {
		return nil, fmt.Errorf("Error expanding the settings file path: %s", err)
	}

	config := Config{
		SettingsFile:   settingsFile,
		SubscriptionID: d.Get("subscription_id").(string),
		Certificate:    []byte(d.Get("certificate").(string)),
	}

	if config.SettingsFile != "" {
		return config.NewClientFromSettingsFile()
	}

	if config.SubscriptionID != "" && len(config.Certificate) > 0 {
		return config.NewClient()
	}

	return nil, fmt.Errorf(
		"Insufficient configuration data. Please specify either a 'settings_file'\n" +
			"or both a 'subscription_id' and 'certificate'.")
}
