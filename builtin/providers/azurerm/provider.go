package azurerm

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"arm_config_file": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				DefaultFunc:  schema.EnvDefaultFunc("ARM_CONFIG_FILE", nil),
				ValidateFunc: validateArmConfigFile,
			},

			"subscription_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
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
			"azurerm_resource_group":  resourceArmResourceGroup(),
			"azurerm_virtual_network": resourceArmVirtualNetwork(),
		},

		ConfigureFunc: providerConfigure,
	}
}

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	ManagementURL string

	ArmConfig string

	SubscriptionID string
	ClientID       string
	ClientSecret   string
	TenantID       string
}

const noConfigError = `Credentials must be provided either via arm_config_file, or via
subscription_id, client_id, client_secret and tenant_id. Please see
the provider documentation for more information on how to obtain these
credentials.`

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		SubscriptionID: d.Get("subscription_id").(string),
		ClientID:       d.Get("client_id").(string),
		ClientSecret:   d.Get("client_secret").(string),
		TenantID:       d.Get("tenant_id").(string),
	}

	// check if credentials file is provided:
	armConfig := d.Get("arm_config_file").(string)
	if armConfig != "" {
		// then, load the settings from that:
		if err := config.readArmSettings(armConfig); err != nil {
			return nil, err
		}
	}

	// then; check whether the ARM credentials were provided:
	if !config.armCredentialsProvided() {
		return nil, fmt.Errorf(noConfigError)
	}

	client, err := config.getArmClient()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func azureRMNormalizeLocation(location interface{}) string {
	input := location.(string)
	return strings.Replace(strings.ToLower(input), " ", "", -1)
}
