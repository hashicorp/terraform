package azurerm

import (
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"subscription_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"azurerm_resource_group":        resourceArmResourceGroup(),
			"azurerm_virtual_network":       resourceArmVirtualNetwork(),
			"azurerm_local_network_gateway": resourceArmLocalNetworkGateway(),
		},

		ConfigureFunc: providerConfigure,
	}
}

// Config is the configuration structure used to instantiate a
// new Azure management client.
type Config struct {
	ManagementURL string

	SubscriptionID string
	ClientID       string
	ClientSecret   string
	TenantID       string
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		SubscriptionID: d.Get("subscription_id").(string),
		ClientID:       d.Get("client_id").(string),
		ClientSecret:   d.Get("client_secret").(string),
		TenantID:       d.Get("tenant_id").(string),
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
