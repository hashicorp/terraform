package azure

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-homedir"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"settings_file": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("AZURE_SETTINGS_FILE", nil),
				ValidateFunc: validateSettingsFile,
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
	config := Config{
		SubscriptionID: d.Get("subscription_id").(string),
		Certificate:    []byte(d.Get("certificate").(string)),
	}

	settings := d.Get("settings_file").(string)

	if settings != "" {
		if ok, _ := isFile(settings); ok {
			settingsFile, err := homedir.Expand(settings)
			if err != nil {
				return nil, fmt.Errorf("Error expanding the settings file path: %s", err)
			}
			publishSettingsContent, err := ioutil.ReadFile(settingsFile)
			if err != nil {
				return nil, fmt.Errorf("Error reading settings file: %s", err)
			}
			config.Settings = publishSettingsContent
		} else {
			config.Settings = []byte(settings)
		}
		return config.NewClientFromSettingsData()
	}

	if config.SubscriptionID != "" && len(config.Certificate) > 0 {
		return config.NewClient()
	}

	return nil, fmt.Errorf(
		"Insufficient configuration data. Please specify either a 'settings_file'\n" +
			"or both a 'subscription_id' and 'certificate'.")
}

func validateSettingsFile(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)

	if value == "" {
		return
	}

	var settings settingsData
	if err := xml.Unmarshal([]byte(value), &settings); err != nil {
		warnings = append(warnings, `
settings_file is not valid XML, so we are assuming it is a file path. This
support will be removed in the future. Please update your configuration to use
${file("filename.publishsettings")} instead.`)
	} else {
		return
	}

	if ok, err := isFile(value); !ok {
		errors = append(errors,
			fmt.Errorf(
				"account_file path could not be read from '%s': %s",
				value,
				err))
	}

	return
}

func isFile(v string) (bool, error) {
	if _, err := os.Stat(v); err != nil {
		return false, err
	}
	return true, nil
}

// settingsData is a private struct used to test the unmarshalling of the
// settingsFile contents, to determine if the contents are valid XML
type settingsData struct {
	XMLName xml.Name `xml:"PublishData"`
}
