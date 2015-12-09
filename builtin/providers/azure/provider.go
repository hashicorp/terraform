package azure

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{

			// ASM-specific fields:
			"settings_file": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				DefaultFunc:  schema.EnvDefaultFunc("AZURE_SETTINGS_FILE", nil),
				ValidateFunc: validateAsmSettingsFile,
				Deprecated:   "Use the 'publish_settings' field instead.",
			},

			"publish_settings": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				DefaultFunc:  schema.EnvDefaultFunc("AZURE_PUBLISH_SETTINGS", nil),
				ValidateFunc: validateAsmPublishSettings,
			},

			// NOTE: subscription_id is a commonly required field for
			// authenticating against both the ASM and ARM-based API's.
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

			// ARM-specific fields:
			"arm_config_file": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				DefaultFunc:  schema.EnvDefaultFunc("ARM_CONFIG_FILE", nil),
				ValidateFunc: validateArmConfigFile,
			},

			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			// ASM-only reources:
			"azure_affinity_group":                    resourceAsmAffinityGroup(),
			"azure_data_disk":                         resourceAsmDataDisk(),
			"azure_sql_database_server":               resourceAsmSqlDatabaseServer(),
			"azure_sql_database_server_firewall_rule": resourceAsmSqlDatabaseServerFirewallRule(),
			"azure_sql_database_service":              resourceAsmSqlDatabaseService(),
			"azure_hosted_service":                    resourceAsmHostedService(),
			"azure_storage_container":                 resourceAsmStorageContainer(),
			"azure_storage_blob":                      resourceAsmStorageBlob(),
			"azure_storage_queue":                     resourceAsmStorageQueue(),

			// both ASM and ARM-based resources:
			"azure_instance":                 resourceAzureInstance(),
			"azure_storage_service":          resourceAzureStorageService(),
			"azure_virtual_network":          resourceAzureVirtualNetwork(),
			"azure_dns_server":               resourceAzureDnsServer(),
			"azure_local_network_connection": resourceAzureLocalNetworkConnection(),
			"azure_security_group":           resourceAzureSecurityGroup(),
			"azure_security_group_rule":      resourceAzureSecurityGroupRule(),

			// ARM-only resources:
			"azure_resource_group": resourceArmResourceGroup(),
			"azure_load_balancer":  resourceArmLoadBalancer(),
			//"azure_application_gateway": resourceArmApplicationGateway(),
			//"azure_availability_set":    resourceArmAvailabilitySet(),
			//"azure_gateway":             resourceArmGateway(),
			//"azure_gateway_connection":  resourceArmGatewayConnection(),
			//"azure_job":                 resourceArmJob(),
			//"azure_job_collection":      resourceArmJobCollection(),
			//"azure_network_interface":   resourceArmNetworkInterface(),
			//"azure_public_ip":           resourceArmPublicIp(),
			//"azure_subnet":              resourceArmSubnet(),
			//"azure_tag":                 resourceArmTag(),
			//"azure_vm_extension_image":  resourceArmVmExtensionImage(),
			//"azure_vm_image":            resourceArmVmImage(),
		},

		ConfigureFunc: providerConfigure,
	}
}

// AzureClient aggregates the clients of both the Azure Service Manager and Azure
// Resource Manager-based APIs.
type AzureClient struct {
	asmClient *AsmClient
	armClient *ArmClient
}

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	// ASM configuration options:
	Settings      string
	Certificate   string
	ManagementURL string

	// SubscriptionID is commonly required by both
	// the ASM and ARM-based APIs.
	SubscriptionID string

	// ARM configuration options:
	ArmConfig string

	ClientID     string
	ClientSecret string
	TenantID     string
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	client := &AzureClient{}
	config := Config{
		SubscriptionID: d.Get("subscription_id").(string),
		Certificate:    d.Get("certificate").(string),
		ClientID:       d.Get("client_id").(string),
		ClientSecret:   d.Get("client_secret").(string),
		TenantID:       d.Get("tenant_id").(string),
	}

	///// ASM setup:
	// first; check for a provided settings file and update
	// the configuration accordingly if provided:
	settingsFile := d.Get("settings_file").(string)
	settingsFileContents := d.Get("publish_settings").(string)
	if settingsFile != "" || settingsFileContents != "" {
		var settings string

		// NOTE: here we intentionally prefer to use the the contents of
		// "publish_settings":
		if settingsFileContents != "" {
			settings = settingsFileContents
		} else {
			settings = settingsFile
		}

		// any errors from readSettings would have been caught at the validate
		// step, so we can avoid handling them now
		conf, _, _ := readAsmSettingsFile(settings)
		config.Settings = conf
	}

	// now; check whether ASM credentials were provided:
	if config.asmCredentialsProvided() {
		// if so; create the ASM client and add it to the AzureClient:
		asmc, err := config.getAsmClient()
		if err != nil {
			return nil, err
		}

		client.asmClient = asmc
	}

	///// ARM setup:
	// check if credentials file provided:
	armConfig := d.Get("arm_config_file").(string)
	if armConfig != "" {
		// then, load the settings from that:
		if err := config.readArmSettings(armConfig); err != nil {
			return nil, err
		}
	}

	// then; check whether the ASM credentials were provided:
	if config.armCredentialsProvided() {
		// if so; create the ARM client:
		armc, err := config.getArmClient()
		if err != nil {
			return nil, err
		}

		client.armClient = armc
	}

	// lastly; check to ensure that the credentials were provided and the
	// client instantiated for at least one of the API settings:
	if client.asmClient == nil && client.armClient == nil {
		return nil, fmt.Errorf(
			"Credentials for at least one of either the ASM or ARM APIs must be provided.",
		)
	}

	return client, nil
}
