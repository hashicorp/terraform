package azure

import (
	"encoding/xml"
	"fmt"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
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
				Deprecated:   "Use the publish_settings field instead",
			},

			"publish_settings": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("AZURE_PUBLISH_SETTINGS", nil),
				ValidateFunc: validatePublishSettings,
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

	publishSettings := d.Get("publish_settings").(string)
	if publishSettings == "" {
		publishSettings = d.Get("settings_file").(string)
	}
	if publishSettings != "" {
		// any errors from readSettings would have been caught at the validate
		// step, so we can avoid handling them now
		settings, _, _ := readSettings(publishSettings)
		config.Settings = settings
		return config.NewClientFromSettingsData()
	}

	if config.SubscriptionID != "" && len(config.Certificate) > 0 {
		return config.NewClient()
	}

	return nil, fmt.Errorf(
		"Insufficient configuration data. Please specify either a 'settings_file'\n" +
			"or both a 'subscription_id' and 'certificate'.")
}

func validateSettingsFile(v interface{}, k string) ([]string, []error) {
	value := v.(string)
	if value == "" {
		return nil, nil
	}

	_, warnings, errors := readSettings(value)
	return warnings, errors
}

func validatePublishSettings(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)
	if value == "" {
		return
	}

	var settings settingsData
	if err := xml.Unmarshal([]byte(value), &settings); err != nil {
		es = append(es, fmt.Errorf("error parsing publish_settings as XML: %s", err))
	}

	return
}

const settingsPathWarnMsg = `
settings_file was provided as a file path. This support
will be removed in the future. Please update your configuration
to use ${file("filename.publishsettings")} instead.`

func readSettings(pathOrContents string) (s []byte, ws []string, es []error) {
	contents, wasPath, err := pathorcontents.Read(pathOrContents)
	if err != nil {
		es = append(es, fmt.Errorf("error reading settings_file: %s", err))
	}
	if wasPath {
		ws = append(ws, settingsPathWarnMsg)
	}

	var settings settingsData
	if err := xml.Unmarshal([]byte(contents), &settings); err != nil {
		es = append(es, fmt.Errorf("error parsing settings_file as XML: %s", err))
	}

	s = []byte(contents)

	return
}

// settingsData is a private struct used to test the unmarshalling of the
// settingsFile contents, to determine if the contents are valid XML
type settingsData struct {
	XMLName xml.Name `xml:"PublishData"`
}
