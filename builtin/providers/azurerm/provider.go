package azurerm

import (
	"fmt"
	"reflect"
	"strings"

	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/mutexkv"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	riviera "github.com/jen20/riviera/azure"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"subscription_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_SUBSCRIPTION_ID", ""),
			},

			"client_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_ID", ""),
			},

			"client_secret": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_CLIENT_SECRET", ""),
			},

			"tenant_id": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ARM_TENANT_ID", ""),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			// These resources use the Azure ARM SDK
			"azurerm_availability_set":          resourceArmAvailabilitySet(),
			"azurerm_cdn_endpoint":              resourceArmCdnEndpoint(),
			"azurerm_cdn_profile":               resourceArmCdnProfile(),
			"azurerm_local_network_gateway":     resourceArmLocalNetworkGateway(),
			"azurerm_network_interface":         resourceArmNetworkInterface(),
			"azurerm_network_security_group":    resourceArmNetworkSecurityGroup(),
			"azurerm_network_security_rule":     resourceArmNetworkSecurityRule(),
			"azurerm_public_ip":                 resourceArmPublicIp(),
			"azurerm_route":                     resourceArmRoute(),
			"azurerm_route_table":               resourceArmRouteTable(),
			"azurerm_storage_account":           resourceArmStorageAccount(),
			"azurerm_storage_blob":              resourceArmStorageBlob(),
			"azurerm_storage_container":         resourceArmStorageContainer(),
			"azurerm_storage_queue":             resourceArmStorageQueue(),
			"azurerm_subnet":                    resourceArmSubnet(),
			"azurerm_template_deployment":       resourceArmTemplateDeployment(),
			"azurerm_virtual_machine":           resourceArmVirtualMachine(),
			"azurerm_virtual_machine_scale_set": resourceArmVirtualMachineScaleSet(),
			"azurerm_virtual_network":           resourceArmVirtualNetwork(),

			// These resources use the Riviera SDK
			"azurerm_dns_a_record":      resourceArmDnsARecord(),
			"azurerm_dns_aaaa_record":   resourceArmDnsAAAARecord(),
			"azurerm_dns_cname_record":  resourceArmDnsCNameRecord(),
			"azurerm_dns_mx_record":     resourceArmDnsMxRecord(),
			"azurerm_dns_ns_record":     resourceArmDnsNsRecord(),
			"azurerm_dns_srv_record":    resourceArmDnsSrvRecord(),
			"azurerm_dns_txt_record":    resourceArmDnsTxtRecord(),
			"azurerm_dns_zone":          resourceArmDnsZone(),
			"azurerm_resource_group":    resourceArmResourceGroup(),
			"azurerm_search_service":    resourceArmSearchService(),
			"azurerm_sql_database":      resourceArmSqlDatabase(),
			"azurerm_sql_firewall_rule": resourceArmSqlFirewallRule(),
			"azurerm_sql_server":        resourceArmSqlServer(),
			"azurerm_simple_lb":         resourceArmSimpleLb(),
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

	validateCredentialsOnce sync.Once
}

func (c *Config) validate() error {
	var err *multierror.Error

	if c.SubscriptionID == "" {
		err = multierror.Append(err, fmt.Errorf("Subscription ID must be configured for the AzureRM provider"))
	}
	if c.ClientID == "" {
		err = multierror.Append(err, fmt.Errorf("Client ID must be configured for the AzureRM provider"))
	}
	if c.ClientSecret == "" {
		err = multierror.Append(err, fmt.Errorf("Client Secret must be configured for the AzureRM provider"))
	}
	if c.TenantID == "" {
		err = multierror.Append(err, fmt.Errorf("Tenant ID must be configured for the AzureRM provider"))
	}

	return err.ErrorOrNil()
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := &Config{
		SubscriptionID: d.Get("subscription_id").(string),
		ClientID:       d.Get("client_id").(string),
		ClientSecret:   d.Get("client_secret").(string),
		TenantID:       d.Get("tenant_id").(string),
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	client, err := config.getArmClient()
	if err != nil {
		return nil, err
	}

	err = registerAzureResourceProvidersWithSubscription(client.rivieraClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func registerProviderWithSubscription(providerName string, client *riviera.Client) error {
	request := client.NewRequest()
	request.Command = riviera.RegisterResourceProvider{
		Namespace: providerName,
	}

	response, err := request.Execute()
	if err != nil {
		return fmt.Errorf("Cannot request provider registration for Azure Resource Manager: %s.", err)
	}

	if !response.IsSuccessful() {
		return fmt.Errorf("Credentials for acessing the Azure Resource Manager API are likely " +
			"to be incorrect, or\n  the service principal does not have permission to use " +
			"the Azure Service Management\n  API.")
	}

	return nil
}

var providerRegistrationOnce sync.Once

// registerAzureResourceProvidersWithSubscription uses the providers client to register
// all Azure resource providers which the Terraform provider may require (regardless of
// whether they are actually used by the configuration or not). It was confirmed by Microsoft
// that this is the approach their own internal tools also take.
func registerAzureResourceProvidersWithSubscription(client *riviera.Client) error {
	var err error
	providerRegistrationOnce.Do(func() {
		// We register Microsoft.Compute during client initialization
		providers := []string{"Microsoft.Network", "Microsoft.Cdn", "Microsoft.Storage", "Microsoft.Sql", "Microsoft.Search", "Microsoft.Resources"}

		var wg sync.WaitGroup
		wg.Add(len(providers))
		for _, providerName := range providers {
			go func(p string) {
				defer wg.Done()
				if innerErr := registerProviderWithSubscription(p, client); err != nil {
					err = innerErr
				}
			}(providerName)
		}
		wg.Wait()
	})

	return err
}

// azureRMNormalizeLocation is a function which normalises human-readable region/location
// names (e.g. "West US") to the values used and returned by the Azure API (e.g. "westus").
// In state we track the API internal version as it is easier to go from the human form
// to the canonical form than the other way around.
func azureRMNormalizeLocation(location interface{}) string {
	input := location.(string)
	return strings.Replace(strings.ToLower(input), " ", "", -1)
}

// armMutexKV is the instance of MutexKV for ARM resources
var armMutexKV = mutexkv.NewMutexKV()

func azureStateRefreshFunc(resourceURI string, client *ArmClient, command riviera.APICall) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		req := client.rivieraClient.NewRequestForURI(resourceURI)
		req.Command = command

		res, err := req.Execute()
		if err != nil {
			return nil, "", fmt.Errorf("Error executing %T command in azureStateRefreshFunc", req.Command)
		}

		var value reflect.Value
		if reflect.ValueOf(res.Parsed).Kind() == reflect.Ptr {
			value = reflect.ValueOf(res.Parsed).Elem()
		} else {
			value = reflect.ValueOf(res.Parsed)
		}

		for i := 0; i < value.NumField(); i++ { // iterates through every struct type field
			tag := value.Type().Field(i).Tag // returns the tag string
			tagValue := tag.Get("mapstructure")
			if tagValue == "provisioningState" {
				return res.Parsed, value.Field(i).Elem().String(), nil
			}
		}

		panic(fmt.Errorf("azureStateRefreshFunc called on structure %T with no mapstructure:provisioningState tag. This is a bug", res.Parsed))
	}
}
